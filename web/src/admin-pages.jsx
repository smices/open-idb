// SPDX-License-Identifier: MIT

import React, { useCallback, useEffect, useState } from 'react';
import {
  App as AntApp,
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Skeleton,
  Space,
  Switch,
  Table,
  Tag,
  Tree,
} from 'antd';
import { Archive, Plus, RefreshCw, Save, Search, Settings } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api } from './lib/api.ts';

const APPLICATION_TYPES = ['oidc_client', 'api_client', 'internal_app'];

function navigate(path) {
  window.location.href = path;
}

function formatDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
}

function errorMessage(error, fallback = 'Request failed') {
  return error instanceof Error ? error.message : fallback;
}

function pickItems(payload, keys = ['items']) {
  for (const key of keys) {
    if (Array.isArray(payload?.[key])) return payload[key];
  }
  return Array.isArray(payload) ? payload : [];
}

export function AdminRouter() {
  const { t } = useTranslation();
  const pathname = window.location.pathname;
  if (pathname === '/admin') return <DashboardPage />;
  if (pathname === '/admin/entities') return <EntitiesPage />;
  if (pathname === '/admin/identity-sources') return <IdentitySourcesPage />;
  if (pathname === '/admin/sync-jobs') return <SyncJobsPage />;
  if (pathname === '/admin/organization') return <OrganizationPage />;
  if (pathname === '/admin/users') return <UsersPage />;
  if (pathname === '/admin/archived-users') return <ArchivedUsersPage />;
  if (/^\/admin\/users\/[^/]+$/.test(pathname)) return <UserDetailPage id={decodeURIComponent(pathname.split('/').pop())} />;
  if (pathname === '/admin/applications') return <ApplicationsPage />;
  if (pathname === '/admin/roles') return <RolesPage />;
  if (pathname === '/admin/audit') return <AuditPage />;
  if (pathname === '/admin/platform') return <PlatformPage />;
  if (pathname === '/admin/admin-users') return <AdminUsersPage />;
  return <Empty description={t('common.notFound')} />;
}

function PageCard({ title, extra, children }) {
  return <Card className="data-card" title={title} extra={extra}>{children}</Card>;
}

function useLoader(loader, deps = []) {
  const { message } = AntApp.useApp();
  const [state, setState] = useState({ loading: true, data: null, error: '' });
  const reload = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: '' }));
    try {
      const data = await loader();
      setState({ loading: false, data, error: '' });
      return data;
    } catch (err) {
      const msg = errorMessage(err);
      setState({ loading: false, data: null, error: msg });
      message.error(msg);
      return null;
    }
  }, deps);
  useEffect(() => {
    reload();
  }, [reload]);
  return { ...state, reload };
}

function DashboardPage() {
  const { t } = useTranslation();
  const { loading, data } = useLoader(() => api.dashboardSummary(), []);
  const metrics = [
    ['users', t('dashboard.users')],
    ['active_users', t('dashboard.activeUsers')],
    ['new_users', t('dashboard.newUsers')],
    ['admin_users', t('dashboard.adminUsers')],
    ['application_activity', t('dashboard.applicationActivity')],
    ['pending_authorization', t('dashboard.pendingAuthorization')],
  ];
  return (
    <div className="page-stack">
      <div className="metric-grid">
        {metrics.map(([key, label]) => <div className="metric-card" key={key}><span>{label}</span><strong>{loading ? '-' : data?.[key] ?? 0}</strong></div>)}
      </div>
      <PageCard title={t('dashboard.title')}>
        {loading ? <Skeleton active /> : <Descriptions column={2} bordered size="small" items={[
          { key: 'sync', label: t('dashboard.syncHealth'), children: data?.sync_health ? t(`dashboard.syncStatus.${data.sync_health}`, data.sync_health) : '-' },
          { key: 'users', label: t('dashboard.totalUsers'), children: data?.users ?? 0 },
        ]} />}
      </PageCard>
    </div>
  );
}

function EntitiesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(() => api.listEntities({ limit: 100 }), []);
  const items = data?.items || [];
  const save = async () => {
    const values = await form.validateFields();
    try {
      if (editing) await api.updateEntity(editing.id, values);
      else await api.createEntity(values);
      message.success(t(editing ? 'common.updateSuccess' : 'common.createSuccess'));
      setOpen(false);
      reload();
    } catch (err) {
      message.error(errorMessage(err));
    }
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setEditing(null); form.resetFields(); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={items} columns={[
        { title: t('entities.name'), dataIndex: 'name' },
        { title: t('entities.slug'), dataIndex: 'slug' },
        { title: t('entities.status'), dataIndex: 'status', render: (v) => <Tag>{t(`entities.status.${v}`, v)}</Tag> },
        { title: t('entities.defaultLocale'), dataIndex: 'default_locale' },
        { title: t('entities.brandName'), dataIndex: 'brand_name' },
        { title: t('entities.createdAt'), dataIndex: 'created_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Button size="small" onClick={() => { setEditing(row); form.setFieldsValue(row); setOpen(true); }}>{t('common.edit')}</Button> },
      ]} />
      <Modal title={editing ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)} destroyOnHidden>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('entities.name')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="slug" label={t('entities.slug')} rules={[{ required: !editing }]}><Input disabled={Boolean(editing)} /></Form.Item>
          <Form.Item name="status" label={t('entities.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`entities.status.${value}`, value) }))} /></Form.Item>
          <Form.Item name="default_locale" label={t('entities.defaultLocale')}><Select options={['zh-CN', 'en-US'].map((value) => ({ value, label: value }))} /></Form.Item>
          <Form.Item name="brand_name" label={t('entities.brandName')}><Input /></Form.Item>
          <Form.Item name="logo_url" label={t('entities.logoUrl')}><Input /></Form.Item>
          <Form.Item name="login_message" label={t('entities.loginMessage')}><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function IdentitySourcesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [configOpen, setConfigOpen] = useState(false);
  const [configForm] = Form.useForm();
  const { loading, data, reload } = useLoader(() => api.listIdentitySources({ limit: 100 }), []);
  const sources = pickItems(data, ['items', 'sources']);
  const loadConfig = async () => {
    setConfigOpen(true);
    try {
      const cfg = await api.getFeishuIdentitySourceConfig();
      configForm.setFieldsValue({ ...cfg, ...(cfg.config || {}) });
    } catch {
      configForm.resetFields();
    }
  };
  const saveConfig = async () => {
    const values = await configForm.validateFields();
    await api.upsertFeishuIdentitySourceConfig({
      display_name: values.display_name || 'Feishu',
      status: values.status || 'active',
      sync_enabled: Boolean(values.sync_enabled),
      config: {
        app_id: values.app_id || '',
        app_secret: values.app_secret || '',
        encrypt_key: values.encrypt_key || '',
        verification_token: values.verification_token || '',
      },
    });
    message.success(t('common.updateSuccess'));
    setConfigOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar">
        <div />
        <Space>
          <Button icon={<Settings size={16} />} onClick={loadConfig}>{t('identitySources.feishuConfigTitle')}</Button>
          <Button type="primary" icon={<Plus size={16} />} onClick={async () => { await api.createIdentitySource({ type: 'feishu', name: 'Feishu', sync_enabled: true }); message.success(t('common.createSuccess')); reload(); }}>{t('identitySources.createFeishuSource')}</Button>
        </Space>
      </div>
      <Table rowKey="id" loading={loading} dataSource={sources} columns={[
        { title: t('entities.name'), dataIndex: 'name' },
        { title: t('identitySources.type'), dataIndex: 'type', render: (v) => t(`identitySources.type.${v}`, v) },
        { title: t('identitySources.status'), dataIndex: 'status', render: (v) => <Tag>{t(`identitySources.status.${v}`, v)}</Tag> },
        { title: t('identitySources.syncEnabled'), dataIndex: 'sync_enabled', render: (v) => <Tag color={v ? 'green' : undefined}>{t(Boolean(v) ? 'common.yes' : 'common.no')}</Tag> },
        { title: t('users.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={async () => { await api.triggerSourceSync(row.id, 'incremental'); message.success(t('identitySources.incrementalSyncStarted')); }}>{t('identitySources.triggerIncremental')}</Button>
          <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteIdentitySource(row.id); message.success(t('common.deleteSuccess')); reload(); }}><Button size="small" danger>{t('common.delete')}</Button></Popconfirm>
        </Space> },
      ]} />
      <Modal title={t('identitySources.feishuConfigTitle')} open={configOpen} onOk={saveConfig} onCancel={() => setConfigOpen(false)} destroyOnHidden>
        <Form form={configForm} layout="vertical">
          <Form.Item name="display_name" label={t('profile.displayName')}><Input /></Form.Item>
          <Form.Item name="status" label={t('identitySources.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`identitySources.status.${value}`, value) }))} /></Form.Item>
          <Form.Item name="sync_enabled" label={t('identitySources.enableSync')} valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="app_id" label={t('identitySources.appId')}><Input /></Form.Item>
          <Form.Item name="app_secret" label={t('identitySources.appSecret')}><Input.Password /></Form.Item>
          <Form.Item name="encrypt_key" label={t('identitySources.encryptKey')}><Input.Password /></Form.Item>
          <Form.Item name="verification_token" label={t('identitySources.verificationToken')}><Input.Password /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function SyncJobsPage() {
  const { t } = useTranslation();
  const { loading, data, reload } = useLoader(() => api.listSyncJobs({ limit: 100 }), []);
  return <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
    { title: t('syncJobs.provider'), dataIndex: 'provider' },
    { title: t('syncJobs.type'), dataIndex: 'type' },
    { title: t('syncJobs.status'), dataIndex: 'status', render: (v) => <Tag>{t(`syncJobs.status.${v}`, v)}</Tag> },
    { title: t('syncJobs.traceId'), dataIndex: 'trace_id', ellipsis: true },
    { title: t('syncJobs.startedAt'), dataIndex: 'started_at', render: formatDate },
    { title: t('syncJobs.finishedAt'), dataIndex: 'finished_at', render: formatDate },
    { title: t('syncJobs.error'), dataIndex: 'error_message', ellipsis: true },
  ]} title={() => <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>} />;
}

function OrganizationPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [treeData, setTreeData] = useState([]);
  const [selected, setSelected] = useState(null);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState('');
  const toNode = (node) => ({ key: `${node.kind}:${node.id}`, title: `${node.name} · ${node.kind}`, raw: node, isLeaf: !node.has_children });
  const loadRoot = async () => {
    setLoading(true);
    try {
      const root = await api.getOrganizationTreeRoot({ limit: 100 });
      setTreeData([toNode(root.root), ...root.children.map(toNode)]);
    } catch (err) {
      message.error(errorMessage(err));
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => { loadRoot(); }, []);
  const onLoadData = async (node) => {
    const raw = node.raw;
    if (!raw || node.children?.length) return;
    const res = await api.listOrganizationTreeChildren({ kind: raw.kind, id: raw.id, limit: 100 });
    node.children = (res.items || []).map(toNode);
    setTreeData([...treeData]);
  };
  const search = async () => {
    if (!query.trim()) return loadRoot();
    const res = await api.searchOrganizationTree({ q: query.trim(), limit: 100 });
    setTreeData((res.items || []).map(toNode));
  };
  return (
    <div className="two-column">
      <Card title={t('organization.organizationTree')} extra={<Space.Compact><Input value={query} onChange={(e) => setQuery(e.target.value)} onPressEnter={search} placeholder={t('organization.search')} /><Button icon={<Search size={16} />} onClick={search} aria-label={t('organization.search')} /></Space.Compact>}>
        {loading ? <Skeleton active /> : <Tree showLine loadData={onLoadData} treeData={treeData} onSelect={(_, info) => setSelected(info.node.raw)} />}
      </Card>
      <Card title={t('directory.details')}>
        {selected ? <Descriptions bordered size="small" column={1} items={Object.entries(selected).map(([key, value]) => ({ key, label: key, children: typeof value === 'boolean' ? String(value) : value || '-' }))} /> : <Empty />}
      </Card>
    </div>
  );
}

function UsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [status, setStatus] = useState('');
  const { loading, data, reload } = useLoader(() => api.listUsers({ status, limit: 100 }), [status]);
  return (
    <div className="page-stack">
      <div className="toolbar">
        <Select allowClear placeholder={t('users.status')} style={{ width: 180 }} value={status || undefined} onChange={(v) => setStatus(v || '')} options={['active', 'disabled', 'inactive'].map((value) => ({ value, label: t(`users.status.${value}`, value) }))} />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
        { title: t('users.username'), dataIndex: 'username' },
        { title: t('users.displayName'), dataIndex: 'display_name' },
        { title: t('users.email'), dataIndex: 'email' },
        { title: t('users.status'), dataIndex: 'lifecycle_status', render: (v) => <Tag>{t(`users.status.${v}`, v)}</Tag> },
        { title: t('users.type'), dataIndex: 'user_type', render: (v) => t(`users.type.${v}`, v) },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={() => navigate(`/admin/users/${encodeURIComponent(row.id)}`)}>{t('common.details')}</Button>
          <Button size="small" onClick={async () => { await (row.lifecycle_status === 'active' ? api.disableUser(row.id) : api.enableUser(row.id)); message.success(t('common.updateSuccess')); reload(); }}>{row.lifecycle_status === 'active' ? t('common.disable') : t('common.enable')}</Button>
        </Space> },
      ]} />
    </div>
  );
}

function ArchivedUsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [filters, setFilters] = useState({});
  const [selectedId, setSelectedId] = useState('');
  const [selected, setSelected] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const { loading, data, reload } = useLoader(() => api.listArchivedUsers({ ...filters, limit: 100 }), [JSON.stringify(filters)]);

  const openDetail = async (id) => {
    setSelectedId(id);
    setDetailLoading(true);
    try {
      const detail = await api.getArchivedUser(id);
      setSelected(detail);
    } catch (err) {
      setSelectedId('');
      message.error(errorMessage(err));
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setSelectedId('');
    setSelected(null);
    setDetailLoading(false);
  };

  const copySelected = async () => {
    if (!selected) return;
    try {
      await navigator.clipboard.writeText(JSON.stringify(selected, null, 2));
      message.success(t('common.copied'));
    } catch (err) {
      message.error(errorMessage(err));
    }
  };

  return (
    <div className="page-stack">
      <div className="toolbar-left">
        <Input
          allowClear
          placeholder={t('archivedUsers.username')}
          prefix={<Archive size={14} />}
          style={{ width: 220 }}
          onChange={(e) => setFilters((current) => ({ ...current, username: e.target.value || undefined }))}
        />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={data?.items || []}
        columns={[
          { title: t('archivedUsers.username'), dataIndex: 'username' },
          { title: t('archivedUsers.displayName'), dataIndex: 'display_name' },
          { title: t('archivedUsers.email'), dataIndex: 'email', render: (value) => value || '-' },
          { title: t('archivedUsers.reason'), dataIndex: 'archive_reason', ellipsis: true, render: (value) => value || '-' },
          { title: t('archivedUsers.archivedAt'), dataIndex: 'archived_at', render: formatDate },
          { title: t('common.actions'), render: (_, row) => <Button size="small" onClick={() => openDetail(row.id)}>{t('archivedUsers.json')}</Button> },
        ]}
      />
      <Drawer
        width={720}
        open={Boolean(selectedId)}
        onClose={closeDetail}
        title={t('archivedUsers.detailTitle')}
        extra={<Button disabled={!selected} onClick={copySelected}>{t('common.copy')}</Button>}
      >
        {detailLoading ? <Skeleton active paragraph={{ rows: 8 }} /> : <pre className="json-box">{selected ? JSON.stringify(selected, null, 2) : ''}</pre>}
      </Drawer>
    </div>
  );
}

function UserDetailPage({ id }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [editOpen, setEditOpen] = useState(false);
  const [bindingOpen, setBindingOpen] = useState(false);
  const [form] = Form.useForm();
  const [bindingForm] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [user, roles, allRoles, sessions, bindings, sources] = await Promise.all([
      api.getUser(id),
      api.getUserRoles(id).catch(() => []),
      api.listRoles({ limit: 200 }).catch(() => ({ items: [] })),
      api.listUserSessions(id, { limit: 50 }).catch(() => ({ items: [] })),
      api.listUserBindings(id).catch(() => []),
      api.listIdentitySources({ limit: 100 }).catch(() => ({ items: [], sources: [] })),
    ]);
    return { user, roles, allRoles: allRoles.items || [], sessions: sessions.items || [], bindings, sources: pickItems(sources, ['items', 'sources']) };
  }, [id]);
  if (loading || !data) return <Skeleton active paragraph={{ rows: 8 }} />;
  const saveUser = async () => {
    const values = await form.validateFields();
    await api.updateUser(id, values);
    message.success(t('common.updateSuccess'));
    setEditOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <PageCard title={data.user.display_name || data.user.username} extra={<Button onClick={() => { form.setFieldsValue(data.user); setEditOpen(true); }}>{t('common.edit')}</Button>}>
        <Descriptions bordered size="small" column={2} items={[
          { key: 'username', label: t('users.username'), children: data.user.username },
          { key: 'email', label: t('users.email'), children: data.user.email || '-' },
          { key: 'phone', label: t('users.phone'), children: data.user.phone || '-' },
          { key: 'status', label: t('users.status'), children: <Tag>{t(`users.status.${data.user.lifecycle_status}`, data.user.lifecycle_status)}</Tag> },
          { key: 'locale', label: t('users.locale'), children: data.user.locale },
          { key: 'updated', label: t('users.updatedAt'), children: formatDate(data.user.updated_at) },
        ]} />
      </PageCard>
      <PageCard title={t('users.roles')} extra={<Select placeholder={t('users.assignRole')} style={{ width: 220 }} options={data.allRoles.map((r) => ({ value: r.id, label: `${r.name} (${r.code})` }))} onChange={async (roleId) => { await api.assignRoleToUser(id, roleId); message.success(t('users.roleAssigned')); reload(); }} />}>
        <Space wrap>{data.roles.map((role) => <Tag key={role.id} closable onClose={(e) => { e.preventDefault(); api.removeRoleFromUser(id, role.id).then(reload); }}>{role.name}</Tag>)}</Space>
      </PageCard>
      <Table title={() => <Space>{t('users.bindings')}<Button size="small" icon={<Plus size={14} />} onClick={() => setBindingOpen(true)}>{t('common.create')}</Button></Space>} rowKey="id" dataSource={data.bindings} columns={[
        { title: t('users.source'), dataIndex: 'source_name' },
        { title: t('users.providerUid'), dataIndex: 'provider_uid' },
        { title: t('users.primaryBinding'), dataIndex: 'is_primary', render: (v) => t(Boolean(v) ? 'common.yes' : 'common.no') },
        { title: t('users.boundAt'), dataIndex: 'bound_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteUserBinding(id, row.id); reload(); }}><Button danger size="small">{t('common.delete')}</Button></Popconfirm> },
      ]} />
      <Table title={() => t('users.sessions')} rowKey="id" dataSource={data.sessions} columns={[
        { title: t('users.loginMethod'), dataIndex: 'login_method' },
        { title: t('users.status'), dataIndex: 'status', render: (v) => <Tag>{t(`users.sessionStatus.${v}`, v)}</Tag> },
        { title: t('users.ip'), dataIndex: 'ip' },
        { title: t('users.createdAt'), dataIndex: 'created_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Button size="small" onClick={async () => { await api.revokeSession(row.id); reload(); }}>{t('users.revokeSession')}</Button> },
      ]} />
      <Modal title={t('common.edit')} open={editOpen} onOk={saveUser} onCancel={() => setEditOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="display_name" label={t('users.displayName')}><Input /></Form.Item>
          <Form.Item name="email" label={t('users.email')}><Input /></Form.Item>
          <Form.Item name="phone" label={t('users.phone')}><Input /></Form.Item>
          <Form.Item name="locale" label={t('users.locale')}><Select options={['zh-CN', 'en-US'].map((value) => ({ value, label: value }))} /></Form.Item>
        </Form>
      </Modal>
      <Modal title={t('users.createBinding')} open={bindingOpen} onOk={async () => { const values = await bindingForm.validateFields(); await api.createUserBinding(id, values); setBindingOpen(false); reload(); }} onCancel={() => setBindingOpen(false)}>
        <Form form={bindingForm} layout="vertical">
          <Form.Item name="source_id" label={t('users.source')} rules={[{ required: true }]}><Select options={data.sources.map((s) => ({ value: s.id, label: s.name }))} /></Form.Item>
          <Form.Item name="directory_user_id" label={t('users.directoryUserId')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="provider_uid" label={t('users.providerUid')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="provider_union_id" label={t('users.providerUnionId')}><Input /></Form.Item>
          <Form.Item name="is_primary" label={t('users.primaryBinding')} valuePropName="checked"><Checkbox /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function ApplicationsPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState(null);
  const [viewing, setViewing] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [apps, clients] = await Promise.all([api.listApplications({ limit: 100 }), api.listOIDCClients({ limit: 200 }).catch(() => ({ clients: [] }))]);
    return { apps: apps.applications || [], clients: clients.clients || [] };
  }, []);
  const oidcClientFor = (app) => {
    const client = data?.clients?.find((item) => item.application_id === app?.id) || null;
    return client;
  };
  const applicationJsonFor = (app) => ({ ...app, oidc_client: oidcClientFor(app) });
  const save = async () => {
    const values = await form.validateFields();
    if (selected) {
      await api.updateApplication(selected.id, { name: values.name, status: values.status });
      const client = data?.clients?.find((item) => item.application_id === selected.id);
      if (selected.type === 'oidc_client' && client) await api.updateOIDCClient(client.id, { redirect_uris: values.redirect_uris, allowed_scopes: values.allowed_scopes, grant_types: values.grant_types, response_types: values.response_types, pkce_required: values.pkce_required, workplace_provider: values.workplace_provider, workplace_app_id: values.workplace_app_id, workplace_app_secret: values.workplace_app_secret, status: values.status });
    }
    else {
      const app = await api.createApplication({ name: values.name, type: values.type || 'oidc_client' });
      if (values.type === 'oidc_client') {
        await api.createOIDCClient({ application_id: app.id, client_id: values.client_id, redirect_uris: values.redirect_uris || [], allowed_scopes: values.allowed_scopes, grant_types: values.grant_types, response_types: values.response_types, pkce_required: values.pkce_required, workplace_provider: values.workplace_provider, workplace_app_id: values.workplace_app_id, workplace_app_secret: values.workplace_app_secret });
      }
    }
    message.success(t('common.updateSuccess'));
    setOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setSelected(null); form.resetFields(); form.setFieldsValue({ type: 'oidc_client', status: 'active', redirect_uris: [], allowed_scopes: ['openid', 'profile'], grant_types: ['authorization_code'], response_types: ['code'], pkce_required: true, workplace_provider: '' }); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={data?.apps || []} columns={[
        { title: t('applications.name'), dataIndex: 'name' },
        { title: t('applications.type'), dataIndex: 'type', render: (v) => t(`applications.type.${v}`, v) },
        { title: t('applications.status'), dataIndex: 'status', render: (v) => <Tag>{t(`applications.status.${v}`, v)}</Tag> },
        { title: t('applications.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={() => setViewing(row)}>{t('common.view')}</Button>
          <Button size="small" onClick={() => { const client = data?.clients?.find((item) => item.application_id === row.id); setSelected(row); form.setFieldsValue({ ...row, ...client }); setOpen(true); }}>{t('common.edit')}</Button>
          <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteApplication(row.id); reload(); }}><Button danger size="small">{t('common.delete')}</Button></Popconfirm>
        </Space> },
      ]} />
      <Modal title={selected ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical"><Form.Item name="name" label={t('applications.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="type" label={t('applications.type')}><Select disabled={Boolean(selected)} options={APPLICATION_TYPES.map((value) => ({ value, label: t(`applications.type.${value}`, value) }))} /></Form.Item><Form.Item name="status" label={t('applications.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`applications.status.${value}`, value) }))} /></Form.Item><Form.Item noStyle shouldUpdate={(prev, next) => prev.type !== next.type}>{() => form.getFieldValue('type') === 'oidc_client' ? <><Form.Item name="client_id" label={t('applications.clientId')}><Input disabled={Boolean(selected)} placeholder={t('applications.autoWhenEmpty')} /></Form.Item><Form.Item name="redirect_uris" label={t('applications.redirectUris')}><Select mode="tags" /></Form.Item><Form.Item name="allowed_scopes" label={t('applications.scopes')}><Select mode="tags" /></Form.Item><Form.Item name="grant_types" label={t('applications.grantTypes')}><Select mode="tags" /></Form.Item><Form.Item name="response_types" label={t('applications.responseTypes')}><Select mode="tags" /></Form.Item><Form.Item name="pkce_required" label={t('applications.pkce')} valuePropName="checked"><Switch /></Form.Item><Form.Item name="workplace_provider" label={t('applications.workplaceProvider')}><Select onChange={(value) => { if (!value) form.setFieldsValue({ workplace_app_id: '', workplace_app_secret: '' }); }} options={[{ value: '', label: t('applications.workplaceProvider.none') }, { value: 'feishu', label: t('applications.workplaceProvider.feishu') }]} /></Form.Item><Form.Item name="workplace_app_id" label={t('applications.workplaceAppId')}><Input /></Form.Item><Form.Item name="workplace_app_secret" label={t('applications.workplaceAppSecret')}><Input.Password /></Form.Item></> : null}</Form.Item></Form>
      </Modal>
      <Modal title={t('common.view')} open={Boolean(viewing)} footer={<Button onClick={async () => { try { await navigator.clipboard.writeText(JSON.stringify(applicationJsonFor(viewing), null, 2)); message.success(t('common.copySuccess')); } catch { message.error(t('common.copyFailed')); } }}>{t('common.copy')}</Button>} onCancel={() => setViewing(null)}><pre className="json-box">{JSON.stringify(applicationJsonFor(viewing), null, 2)}</pre></Modal>
    </div>
  );
}

function RolesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [roleOpen, setRoleOpen] = useState(false);
  const [permOpen, setPermOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState(null);
  const [roleForm] = Form.useForm();
  const [permForm] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [roles, permissions] = await Promise.all([api.listRoles({ limit: 200 }), api.listPermissions({ limit: 200 })]);
    return { roles: roles.items || [], permissions: permissions.items || [] };
  }, []);
  const saveRole = async () => {
    const values = await roleForm.validateFields();
    if (selectedRole) await api.updateRole(selectedRole.id, values);
    else await api.createRole(values);
    message.success(t('common.updateSuccess'));
    setRoleOpen(false);
    reload();
  };
  const savePermission = async () => {
    const values = await permForm.validateFields();
    await api.createPermission(values);
    message.success(t('common.createSuccess'));
    setPermOpen(false);
    reload();
  };
  return (
    <div className="two-column">
      <Card title={t('roles.title')} extra={<Button icon={<Plus size={16} />} onClick={() => { setSelectedRole(null); roleForm.resetFields(); setRoleOpen(true); }}>{t('common.create')}</Button>}>
        <Table rowKey="id" loading={loading} dataSource={data?.roles || []} pagination={false} columns={[
          { title: t('roles.name'), dataIndex: 'name' },
          { title: t('roles.code'), dataIndex: 'code' },
          { title: t('common.actions'), render: (_, row) => <Space><Button size="small" onClick={() => { setSelectedRole(row); roleForm.setFieldsValue(row); setRoleOpen(true); }}>{t('common.edit')}</Button><RolePermissionEditor role={row} permissions={data?.permissions || []} /></Space> },
        ]} />
      </Card>
      <Card title={t('roles.permissions')} extra={<Button icon={<Plus size={16} />} onClick={() => setPermOpen(true)}>{t('common.create')}</Button>}>
        <Table rowKey="id" loading={loading} dataSource={data?.permissions || []} pagination={{ pageSize: 8 }} columns={[{ title: t('roles.code'), dataIndex: 'code' }, { title: t('roles.name'), dataIndex: 'name' }, { title: t('roles.type'), dataIndex: 'type' }]} />
      </Card>
      <Modal title={t('roles.role')} open={roleOpen} onOk={saveRole} onCancel={() => setRoleOpen(false)}><Form form={roleForm} layout="vertical"><Form.Item name="name" label={t('roles.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="code" label={t('roles.code')} rules={[{ required: !selectedRole }]}><Input disabled={Boolean(selectedRole)} /></Form.Item><Form.Item name="description" label={t('roles.description')}><Input.TextArea /></Form.Item></Form></Modal>
      <Modal title={t('roles.permission')} open={permOpen} onOk={savePermission} onCancel={() => setPermOpen(false)}><Form form={permForm} layout="vertical"><Form.Item name="code" label={t('roles.code')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="name" label={t('roles.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="type" label={t('roles.type')} rules={[{ required: true }]}><Select options={['api', 'menu', 'data'].map((value) => ({ value, label: t(`roles.permissionType.${value}`, value) }))} /></Form.Item></Form></Modal>
    </div>
  );
}

function RolePermissionEditor({ role, permissions }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState([]);
  const load = async () => {
    const current = await api.listRolePermissions(role.id).catch(() => []);
    setSelected(current.map((p) => p.id));
    setOpen(true);
  };
  const save = async () => {
    const current = await api.listRolePermissions(role.id).catch(() => []);
    const currentIds = new Set(current.map((p) => p.id));
    const nextIds = new Set(selected);
    await Promise.all([
      ...selected.filter((id) => !currentIds.has(id)).map((id) => api.assignPermissionToRole(role.id, id)),
      ...current.filter((p) => !nextIds.has(p.id)).map((p) => api.removePermissionFromRole(role.id, p.id)),
    ]);
    message.success(t('roles.permissionSaveSuccess'));
    setOpen(false);
  };
  return <><Button size="small" onClick={load}>{t('roles.permissions')}</Button><Modal title={t('roles.permissionDialogTitle', { name: role.name })} open={open} onOk={save} onCancel={() => setOpen(false)}><Select mode="multiple" style={{ width: '100%' }} value={selected} onChange={setSelected} options={permissions.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))} /></Modal></>;
}

function AuditPage() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState({});
  const [selected, setSelected] = useState(null);
  const { loading, data, reload } = useLoader(() => api.listAuditLogs({ ...filters, limit: 100 }), [JSON.stringify(filters)]);
  return (
    <div className="page-stack">
      <div className="toolbar-left">
        <Input placeholder={t('audit.action')} style={{ width: 180 }} onChange={(e) => setFilters((f) => ({ ...f, action: e.target.value || undefined }))} />
        <Input placeholder={t('audit.resourceType')} style={{ width: 180 }} onChange={(e) => setFilters((f) => ({ ...f, resource_type: e.target.value || undefined }))} />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
        { title: t('audit.action'), dataIndex: 'action', render: (v) => t(`audit.action.${v}`, v) },
        { title: t('audit.actor'), dataIndex: 'actor_type', render: (v) => t(`audit.actor.${v}`, v) },
        { title: t('audit.resourceType'), render: (_, row) => `${t(`audit.resource.${row.resource_type}`, row.resource_type)}/${row.resource_id}` },
        { title: t('audit.ip'), dataIndex: 'ip' },
        { title: t('audit.time'), dataIndex: 'created_at', render: formatDate },
        { title: t('audit.traceId'), dataIndex: 'trace_id', ellipsis: true },
        { title: t('audit.details'), render: (_, row) => <Button size="small" onClick={() => setSelected(row)}>JSON</Button> },
      ]} />
      <Drawer width={640} open={Boolean(selected)} onClose={() => setSelected(null)} title={t('audit.detailTitle')}><pre className="json-box">{JSON.stringify(selected, null, 2)}</pre></Drawer>
    </div>
  );
}

function PlatformPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [form] = Form.useForm();
  const { loading } = useLoader(async () => {
    const branding = await api.getAdminPlatformBranding();
    form.setFieldsValue(branding);
    return branding;
  }, []);
  const save = async () => {
    const values = await form.validateFields();
    await api.updatePlatformBranding(values);
    message.success(t('common.updateSuccess'));
  };
  return <Card className="data-card" title={t('platform.title')} loading={loading} extra={<Button type="primary" icon={<Save size={16} />} onClick={save}>{t('common.save')}</Button>}><Form form={form} layout="vertical"><Form.Item name="platform_name" label={t('platform.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="logo_url" label={t('platform.logoUrl')}><Input /></Form.Item><Form.Item name="favicon_url" label={t('platform.faviconUrl')}><Input /></Form.Item><Form.Item name="title_suffix" label={t('platform.titleSuffix')}><Input /></Form.Item></Form></Card>;
}

function AdminUsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [passwordMode, setPasswordMode] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [admins, roles, entities] = await Promise.all([api.listAdminUsers(), api.listAdminRoles(), api.listEntities({ limit: 200 }).catch(() => ({ items: [] }))]);
    return { admins: admins.items || [], roles, entities: entities.items || [] };
  }, []);
  const save = async () => {
    const values = await form.validateFields();
    if (passwordMode) await api.setAdminUserPassword(editing.id, values.password);
    else if (editing) await api.updateAdminUser(editing.id, values);
    else await api.createAdminUser(values);
    message.success(t('common.updateSuccess'));
    setOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setEditing(null); setPasswordMode(false); form.resetFields(); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={data?.admins || []} columns={[
        { title: t('adminUsers.account'), dataIndex: 'username' },
        { title: t('adminUsers.displayName'), dataIndex: 'display_name' },
        { title: t('adminUsers.role'), dataIndex: 'role', render: (v) => <Tag>{v}</Tag> },
        { title: t('adminUsers.status'), dataIndex: 'status', render: (v) => <Tag>{t(`adminUsers.status.${v}`, v)}</Tag> },
        { title: t('adminUsers.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space><Button size="small" onClick={() => { setEditing(row); setPasswordMode(false); form.setFieldsValue(row); setOpen(true); }}>{t('common.edit')}</Button><Button size="small" onClick={() => { setEditing(row); setPasswordMode(true); form.resetFields(); setOpen(true); }}>{t('adminUsers.changePassword')}</Button><Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteAdminUser(row.id); reload(); }}><Button size="small" danger>{t('common.delete')}</Button></Popconfirm></Space> },
      ]} />
      <Modal title={passwordMode ? t('adminUsers.changePassword') : editing ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical">
          {!passwordMode ? <><Form.Item name="username" label={t('adminUsers.loginAccount')} rules={[{ required: !editing }]}><Input disabled={Boolean(editing)} /></Form.Item><Form.Item name="display_name" label={t('adminUsers.displayName')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="email" label={t('adminUsers.email')}><Input /></Form.Item><Form.Item name="role" label={t('adminUsers.role')} rules={[{ required: true }]}><Select options={(data?.roles || []).map((r) => ({ value: r.value, label: r.label }))} /></Form.Item><Form.Item name="entity_id" label={t('adminUsers.company')}><Select allowClear options={(data?.entities || []).map((e) => ({ value: e.id, label: e.name }))} /></Form.Item><Form.Item name="status" label={t('adminUsers.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`adminUsers.status.${value}`, value) }))} /></Form.Item></> : null}
          {(!editing || passwordMode) ? <Form.Item name="password" label={t('login.password')} rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item> : null}
        </Form>
      </Modal>
    </div>
  );
}
