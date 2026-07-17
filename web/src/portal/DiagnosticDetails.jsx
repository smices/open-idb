import { Button, Space, Typography } from 'antd';
import { Copy } from 'lucide-react';
import { useState } from 'react';
import { copyDiagnostic, diagnosticCode, formatDiagnostic } from '../lib/diagnostics.js';

export function DiagnosticDetails({ error, route, t }) {
  const [copied, setCopied] = useState(false);
  const [copyFailed, setCopyFailed] = useState(false);
  const diagnostic = formatDiagnostic({
    code: diagnosticCode(error, 'portal_application_list_failed'),
    stage: 'portal_applications',
    route,
    error,
  });

  const copy = async () => {
    try {
      await copyDiagnostic(diagnostic);
      setCopied(true);
      setCopyFailed(false);
    } catch {
      setCopyFailed(true);
      setCopied(false);
    }
  };

  return (
    <Space direction="vertical" size={8}>
      <Typography.Paragraph type="secondary" copyable={{ text: diagnostic }} style={{ margin: 0 }}>
        {t('diagnostics.safeDescription')}
      </Typography.Paragraph>
      <Button size="small" icon={<Copy size={14} />} onClick={copy} aria-label={t('diagnostics.copy')}>
        {copied ? t('diagnostics.copied') : t('diagnostics.copy')}
      </Button>
      {copyFailed ? <Typography.Text type="danger">{t('diagnostics.copyFailed')}</Typography.Text> : null}
    </Space>
  );
}
