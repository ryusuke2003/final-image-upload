import { useState } from 'react';

type UploadUrlResponse = { url: string; key: string; headers?: Record<string, string> };

export default function App() {
  const [file, setFile] = useState<File | null>(null);
  const [status, setStatus] = useState<string>('');
  const [imageUrl, setImageUrl] = useState<string>('');

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFile(e.target.files?.[0] ?? null);
    setStatus('');
    setImageUrl('');
  };

  const upload = async () => {
    if (!file) {
      setStatus('ファイルを選択してください');
      return;
    }
    if (!file.type.startsWith('image/')) {
      setStatus('画像ファイルのみアップロードできます');
      return;
    }

    setStatus('署名URLを取得中…');

    // 1) 署名付きURLを取得（SDK発行, 必要ならヘッダも返る）
    const res1 = await fetch('/api/upload-url', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        filename: file.name,
        contentType: file.type // 署名に含めたい場合
      })
    });

    if (!res1.ok) {
      const err = await res1.text();
      setStatus(`URL発行エラー: ${err}`);
      return;
    }

    const { url, key, headers } = (await res1.json()) as UploadUrlResponse;

    setStatus('S3へ直接PUT中…');

    // 2) 署名時に含めたヘッダはPUTでも完全一致させる
    const putRes = await fetch(url, {
      method: 'PUT',
      headers: headers ?? undefined,
      body: file
    });

    if (!putRes.ok) {
      setStatus(`アップロード失敗: HTTP ${putRes.status}`);
      return;
    }

    const eTagHeader = putRes.headers.get('ETag') || putRes.headers.get('Etag') || undefined;
    const eTag = eTagHeader ? eTagHeader.replaceAll('"', '') : undefined;

    const publicUrl = url.split('?')[0];
    setStatus('メタデータ保存中…');

    // 3) メタデータを保存（PostgreSQL）
    const res2 = await fetch('/api/images', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        key,
        url: publicUrl,
        contentType: file.type,
        size: file.size,
        eTag
      })
    });

    if (!res2.ok) {
      const err = await res2.text();
      setStatus(`メタデータ保存エラー: ${err}`);
      return;
    }

    setImageUrl(publicUrl);
    setStatus('アップロード完了！');
  };

  return (
    <main style={{ maxWidth: 560, margin: '40px auto', fontFamily: 'system-ui, sans-serif' }}>
      <h1>画像アップロード </h1>
      <input type="file" accept="image/*" onChange={onFileChange} />
      <div style={{ marginTop: 12 }}>
        <button onClick={upload} disabled={!file}>アップロード</button>
      </div>
      {status && <p style={{ marginTop: 12 }}>{status}</p>}
      {imageUrl && (
        <div style={{ marginTop: 12 }}>
          <p>アップロード先:</p>
          <a href={imageUrl} target="_blank" rel="noreferrer">{imageUrl}</a>
          <div style={{ marginTop: 12 }}>
            {/* バケットが公開読み取りでない場合は直接表示不可 */}
            <img src={imageUrl} alt="uploaded" style={{ maxWidth: '100%', height: 'auto' }} />
          </div>
        </div>
      )}
    </main>
  );
}
