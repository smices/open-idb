import { App as AntApp, Button, Card, Form, Input } from 'antd';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../lib/api.ts';

function errorMessage(error, fallback = 'Request failed') {
  return error instanceof Error ? error.message : fallback;
}

function runAction(message, action) {
  return Promise.resolve()
    .then(action)
    .catch((error) => {
      if (!Array.isArray(error?.errorFields)) message.error(errorMessage(error));
      return null;
    });
}

export function ProfilePage({ user }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();

  useEffect(() => {
    profileForm.resetFields();
    profileForm.setFieldsValue(user);
  }, [profileForm, user]);

  const saveProfile = () => runAction(message, async () => {
    const values = await profileForm.validateFields();
    await api.updateMe({ display_name: values.display_name });
    message.success(t('common.updateSuccess'));
  });
  const savePassword = () => runAction(message, async () => {
    const values = await passwordForm.validateFields();
    await api.updatePassword(values);
    passwordForm.resetFields();
    message.success(t('common.updateSuccess'));
  });

  return (
    <div className="two-column">
      <Card title={t('profile.title')} extra={<Button onClick={saveProfile}>{t('common.save')}</Button>}>
        <Form form={profileForm} layout="vertical"><Form.Item name="display_name" label={t('profile.displayName')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="username" label={t('users.username')}><Input disabled /></Form.Item><Form.Item name="email" label={t('users.email')}><Input disabled /></Form.Item></Form>
      </Card>
      <Card title={t('profile.updatePassword')} extra={<Button onClick={savePassword}>{t('common.save')}</Button>}>
        <Form form={passwordForm} layout="vertical"><Form.Item name="current_password" label={t('profile.currentPassword')} rules={[{ required: true }]}><Input.Password /></Form.Item><Form.Item name="new_password" label={t('profile.newPassword')} rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item></Form>
      </Card>
    </div>
  );
}
