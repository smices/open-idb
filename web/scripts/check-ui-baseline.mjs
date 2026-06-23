import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const packageJson = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
const appCss = readFileSync(join(root, 'src/app.css'), 'utf8');

const dependencies = {
  ...packageJson.dependencies,
  ...packageJson.devDependencies,
};

const failures = [];

for (const packageName of Object.keys(dependencies)) {
  if (packageName.includes('shadcn')) {
    failures.push(`Unexpected shadcn dependency: ${packageName}`);
  }
}

const skeletonThemeImports = appCss
  .split('\n')
  .filter((line) => line.startsWith("@import '@skeletonlabs/skeleton/themes/"));

if (skeletonThemeImports.length !== 1) {
  failures.push(`Expected exactly one Skeleton theme import, got: ${skeletonThemeImports.length}`);
}

if (!appCss.includes('--idb-radius-card')) {
  failures.push('Missing IdBridge design tokens in app.css');
}

if (failures.length) {
  console.error(failures.join('\n'));
  process.exit(1);
}

console.log('UI baseline checks passed.');
