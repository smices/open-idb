import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const repoRoot = join(root, '..');
const packageJson = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
const appCss = readFileSync(join(root, 'src/styles.css'), 'utf8');
const appJs = readFileSync(join(root, 'src/main.jsx'), 'utf8');
const adminPagesJs = readFileSync(join(root, 'src/admin-pages.jsx'), 'utf8');
const apiTs = readFileSync(join(root, 'src/lib/api.ts'), 'utf8');
const i18nJs = readFileSync(join(root, 'src/i18n/index.js'), 'utf8');
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

const reactMajor = Number.parseInt(String(dependencies.react || '').match(/\d+/)?.[0] || '0', 10);
const antdMajor = Number.parseInt(String(dependencies.antd || '').match(/\d+/)?.[0] || '0', 10);
if (reactMajor >= 19 && antdMajor === 5) {
  if (!dependencies['@ant-design/v5-patch-for-react-19']) {
    failures.push('React 19 with Ant Design 5 requires the official compatibility package');
  }
  if (!appJs.includes("import '@ant-design/v5-patch-for-react-19';")) {
    failures.push('React 19 compatibility package must be loaded by the main application entry');
  }
}

for (const command of ['test:unit', 'typecheck', 'check:ui', 'build']) {
  if (!String(packageJson.scripts?.check || '').includes(`npm run ${command}`)) {
    failures.push(`Web check script must include npm run ${command}`);
  }
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

if (devWebLocal.includes('VITE_API_TARGET=')) {
  failures.push('Local web startup must keep the API target server-only so browser requests use the Vite proxy');
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
if (!appJs.includes('locale={themeState.language') || !appJs.includes('antdZhCN') || !appJs.includes('antdEnUS')) {
  failures.push('Ant Design locale must follow the selected application language');
}
if (!appJs.includes('mobile-menu-button') || !appCss.includes('.mobile-sidebar-backdrop')) {
  failures.push('Admin shell must provide a mobile navigation control and overlay');
}
if (!/\.nav-menu\s*\{[\s\S]*?min-height:\s*0;[\s\S]*?overflow-y:\s*auto;[\s\S]*?\}/.test(appCss)) {
  failures.push('Admin navigation must remain scrollable on short viewports');
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

const applicationTypes = ['oidc_client', 'api_client', 'internal_app'];
const applicationTypeDeclaration = `const APPLICATION_TYPES = ['oidc_client', 'api_client', 'internal_app'];`;
const applicationTypeUnion = `export type ApplicationType = 'oidc_client' | 'api_client' | 'internal_app';`;

if (!adminPagesJs.includes(applicationTypeDeclaration)) {
  failures.push('Applications page must declare the canonical application type list');
}
if (!adminPagesJs.includes('offset: (page - 1) * APPLICATION_PAGE_SIZE') || !adminPagesJs.includes('total: data?.total')) {
  failures.push('Applications page must paginate through the complete server-side result set');
}
if (!adminPagesJs.includes("type: values.type || 'oidc_client'")) {
  failures.push('Applications page must default new applications to oidc_client');
}
if (!adminPagesJs.includes("workplaceProvider === 'feishu'") || !adminPagesJs.includes('required: requireWorkplaceConfig')) {
  failures.push('Feishu workplace selection must require its application ID and secret');
}
if (/on(?:Click|Confirm|Ok|Change)=\{async/.test(`${appJs}\n${adminPagesJs}`)) {
  failures.push('Admin mutation handlers must route failures through shared error handling');
}
if (!adminPagesJs.includes('const requestIdRef = useRef(0)') || !adminPagesJs.includes('requestId === requestIdRef.current')) {
  failures.push('Async page loaders must ignore stale responses');
}
if (!/const onLoadData = async[\s\S]*message\.error\(errorMessage\(error\)\);[\s\S]*throw error;/.test(adminPagesJs)) {
  failures.push('Organization lazy loading must report errors without converting rejection to success');
}
if (!adminPagesJs.includes('children: root.children.map(toNode)')) {
  failures.push('Organization tree root must retain its direct children as nested tree data.');
}
if (!adminPagesJs.includes('replaceTreeNodeChildren(current, node.key, children)')) {
  failures.push('Organization tree loader must immutably attach loaded children to the selected node.');
}
if (adminPagesJs.includes('<Checkbox') && !/import\s*\{[\s\S]*?\bCheckbox\b[\s\S]*?\}\s*from\s*['"]antd['"]/.test(adminPagesJs)) {
  failures.push('Admin pages must import Checkbox from antd before rendering user binding forms.');
}
if (adminPagesJs.includes('setConfigOpen(true);\n    try')) {
  failures.push('Identity source configuration must not open before its existing values load successfully');
}
if (!adminPagesJs.includes("api.triggerSourceSync(row.id, 'full')") || !adminPagesJs.includes("t('identitySources.triggerFull')")) {
  failures.push('Identity sources page must keep the guarded full-sync recovery action');
}
if (!apiTs.includes('/sync/${mode}') || !apiTs.includes('triggerSourceSync:')) {
  failures.push('Frontend API must keep the identity-source sync trigger endpoint');
}
if (!/configForm\.resetFields\(\);\s*configForm\.setFieldsValue/.test(adminPagesJs)) {
  failures.push('Identity source configuration reload must clear cancelled form values before applying server state');
}
if (adminPagesJs.includes('api.listRolePermissions(role.id).catch')) {
  failures.push('Role permission read failures must not be treated as an empty permission set');
}
if (!adminPagesJs.includes('setEditing(row); setPasswordMode(false); form.resetFields(); form.setFieldsValue(row);')) {
  failures.push('Admin user editing must clear values left by the previously selected account');
}
if (!appJs.includes('const saveProfile = () => runAction') || !appJs.includes('const savePassword = () => runAction')) {
  failures.push('Profile mutations must surface API and validation failures');
}
if (!apiTs.includes(applicationTypeUnion) || !apiTs.includes('payload: ApplicationWriteRequest')) {
  failures.push('Frontend API must expose the canonical ApplicationType contract');
}
for (const method of ['deleteArchivedUser:', 'clearArchivedUsers:', 'deleteAuditLog:', 'clearAuditLogs:']) {
  if (!apiTs.includes(method)) {
    failures.push(`Frontend API must expose destructive admin method: ${method}`);
  }
}
for (const translationKey of [
  'archivedUsers.deleteConfirm',
  'archivedUsers.clearConfirm',
  'audit.deleteConfirm',
  'audit.clearConfirm',
]) {
  if (i18nJs.split(`'${translationKey}':`).length - 1 !== 2) {
    failures.push(`Missing bilingual destructive-action label: ${translationKey}`);
  }
}
if (!adminPagesJs.includes('api.deleteArchivedUser(id)') || !adminPagesJs.includes('deleteArchivedUser(row.id)') || !adminPagesJs.includes('api.clearArchivedUsers()')) {
  failures.push('Archived users page must expose single-delete and clear-all controls');
}
if (!adminPagesJs.includes('api.deleteAuditLog(id)') || !adminPagesJs.includes('deleteAuditLog(row.id)') || !adminPagesJs.includes('api.clearAuditLogs()')) {
  failures.push('Audit page must expose single-delete and clear-all controls');
}
for (const appType of applicationTypes) {
  const key = `'applications.type.${appType}':`;
  if (i18nJs.split(key).length - 1 !== 2) {
    failures.push(`Missing bilingual application type label: ${appType}`);
  }
}
for (const obsoleteType of ['oidc', 'saml', 'custom']) {
  if (i18nJs.includes(`'applications.type.${obsoleteType}':`)) {
    failures.push(`Obsolete application type translation remains: ${obsoleteType}`);
  }
}

if (failures.length) {
  console.error(failures.join('\n'));
  process.exit(1);
}

console.log('UI baseline checks passed.');
