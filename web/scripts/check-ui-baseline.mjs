import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const packageJson = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
const appCss = readFileSync(join(root, 'src/styles.css'), 'utf8');
const appJs = readFileSync(join(root, 'src/main.jsx'), 'utf8');

const dependencies = {
  ...packageJson.dependencies,
  ...packageJson.devDependencies,
};

const failures = [];

for (const packageName of Object.keys(dependencies)) {
  if (packageName.includes('shadcn')) {
    failures.push(`Unexpected shadcn dependency: ${packageName}`);
  }
  if (packageName.includes('svelte') || packageName.includes('skeleton')) {
    failures.push(`Unexpected legacy Svelte dependency: ${packageName}`);
  }
}

if (!dependencies.antd) {
  failures.push('Missing Ant Design dependency');
}

if (!appCss.includes('var(--ant-color-bg-layout)')) {
  failures.push('Missing Ant Design token usage in app.css');
}

if (!appCss.includes('min-height: 100dvh')) {
  failures.push('Missing responsive dvh shell sizing');
}

const allowedUiLiterals = new Set([
  'JSON',
]);
const hardcodedUiPatterns = [
  /title:\s*['"]([A-Z][^'"]*)['"]/g,
  /label=["']([A-Z][^"']*)["']/g,
  /placeholder=["']([A-Z][^"']*)["']/g,
  />\s*([A-Z][A-Za-z /-]{2,})\s*</g,
  /message\.success\(['"]([A-Z][^'"]*)['"]\)/g,
];

for (const pattern of hardcodedUiPatterns) {
  for (const match of appJs.matchAll(pattern)) {
    const text = match[1].trim();
    if (!allowedUiLiterals.has(text)) {
      failures.push(`Hardcoded English UI text: ${text}`);
    }
  }
}

if (failures.length) {
  console.error(failures.join('\n'));
  process.exit(1);
}

console.log('UI baseline checks passed.');
