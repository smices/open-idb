import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const repoRoot = join(root, '..');
const packageJson = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
const appCss = readFileSync(join(root, 'src/styles.css'), 'utf8');
const appJs = readFileSync(join(root, 'src/main.jsx'), 'utf8');
const adminPagesJs = readFileSync(join(root, 'src/admin-pages.jsx'), 'utf8');
const bootJs = readFileSync(join(root, 'src/boot.js'), 'utf8');
const workplaceContinueJs = readFileSync(join(root, 'src/workplace-continue.js'), 'utf8');
const devWebLocal = readFileSync(join(repoRoot, 'scripts/dev-web-local.sh'), 'utf8');
const webContract = readFileSync(join(repoRoot, 'docs/web-react-antd-contract.md'), 'utf8');

const dependencies = {
  ...packageJson.dependencies,
  ...packageJson.devDependencies,
};

const failures = [];

function collectFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    return entry.isDirectory() ? collectFiles(path) : [path];
  });
}

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

for (const filePath of collectFiles(join(root, 'src'))) {
  if (filePath.endsWith('.svelte')) {
    failures.push(`Unexpected legacy Svelte source file: ${filePath}`);
  }
}

for (const [label, source] of [
  ['scripts/dev-web-local.sh', devWebLocal],
  ['web/src/boot.js', bootJs],
  ['web/src/main.jsx', appJs],
]) {
  if (source.includes('Svelte web') || source.includes('warn_legacy_react_frontend')) {
    failures.push(`Legacy Svelte migration wording remains in ${label}`);
  }
}

if (!webContract.includes('workplace-continue.js')) {
  failures.push('Missing documented lightweight workplace entry exception in web frontend contract');
}

if (!bootJs.includes("import('./workplace-continue.js')")) {
  failures.push('Missing lightweight workplace continue dynamic entry');
}

if (!appJs.includes("React.lazy(() => import('./admin-pages.jsx')")) {
  failures.push('Admin pages must be route-lazy loaded from the main React entry');
}

for (const forbidden of ['react', 'react-dom', 'antd', 'react-i18next', './i18n/index.js']) {
  if (workplaceContinueJs.includes(`from '${forbidden}'`) || workplaceContinueJs.includes(`from "${forbidden}"`) || workplaceContinueJs.includes(`import '${forbidden}'`) || workplaceContinueJs.includes(`import "${forbidden}"`)) {
    failures.push(`Lightweight workplace continue entry must not import ${forbidden}`);
  }
}

if (!workplaceContinueJs.includes("const THEME_KEY = 'idb-theme-mode'")) {
  failures.push('Lightweight workplace continue entry must follow shared theme preference key');
}

if (!workplaceContinueJs.includes('const MESSAGES =')) {
  failures.push('Lightweight workplace continue entry must keep local small-bundle i18n messages');
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

for (const [label, source] of [
  ['web/src/main.jsx', appJs],
  ['web/src/admin-pages.jsx', adminPagesJs],
]) {
  for (const pattern of hardcodedUiPatterns) {
    for (const match of source.matchAll(pattern)) {
      const text = match[1].trim();
      if (!allowedUiLiterals.has(text)) {
        failures.push(`Hardcoded English UI text in ${label}: ${text}`);
      }
    }
  }
}

if (failures.length) {
  console.error(failures.join('\n'));
  process.exit(1);
}

console.log('UI baseline checks passed.');
