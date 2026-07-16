import { Alert, Avatar, Button, Card, Empty, Skeleton, Space, Tag, Typography } from 'antd';
import { AppWindow, ExternalLink, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { usePortalApplications } from './usePortalApplications.js';

function applicationTypeLabel(application, t) {
  const key = `applications.type.${application.type}`;
  const translated = t(key);
  return translated === key ? application.type : translated;
}

function ApplicationCard({ application, t }) {
  const title = application.name || application.id;

  return (
    <Card className="portal-application-card">
      <Space align="start" size={12}>
        <Avatar shape="square" size={44} src={application.logo_url} icon={<AppWindow size={20} />} />
        <div className="portal-application-card-heading">
          <Typography.Title level={4}>{title}</Typography.Title>
          <Tag>{applicationTypeLabel(application, t)}</Tag>
        </div>
      </Space>
      {application.description ? <Typography.Paragraph className="portal-application-description">{application.description}</Typography.Paragraph> : null}
      {application.entry_url ? (
        <Button type="primary" icon={<ExternalLink size={16} />} href={application.entry_url} target="_blank" rel="noreferrer">
          {t('portal.openApplication')}
        </Button>
      ) : null}
    </Card>
  );
}

export function PortalHomePage() {
  const { t } = useTranslation();
  const { applications, error, loading, reload } = usePortalApplications();

  if (loading) {
    return <Skeleton active paragraph={{ rows: 8 }} />;
  }

  if (error) {
    return (
      <Alert
        type="error"
        showIcon
        message={t('portal.fetchFailed')}
        action={<Button size="small" icon={<RefreshCw size={14} />} onClick={reload}>{t('common.retry')}</Button>}
      />
    );
  }

  if (!applications.length) {
    return <Empty description={t('portal.empty')} />;
  }

  return (
    <section aria-label={t('portal.title')}>
      <div className="portal-applications-grid">
        {applications.map((application) => <ApplicationCard key={application.id} application={application} t={t} />)}
      </div>
    </section>
  );
}
