import { copyFile, mkdir } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

const assets = [
  {
    from: 'node_modules/alpinejs/dist/cdn.min.js',
    to: 'internal/web/vendor/alpinejs/cdn.min.js',
  },
  {
    from: 'node_modules/echarts/dist/echarts.min.js',
    to: 'internal/web/vendor/echarts/echarts.min.js',
  },
];

for (const asset of assets) {
  const source = join(root, asset.from);
  const target = join(root, asset.to);
  await mkdir(dirname(target), { recursive: true });
  await copyFile(source, target);
  console.log(`${asset.from} -> ${asset.to}`);
}
