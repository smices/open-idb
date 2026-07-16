import { Avatar, Card, Descriptions } from 'antd';
import { useTranslation } from 'react-i18next';

export function ProfilePage({ user }) {
  const { t } = useTranslation();
  const value = (field) => user?.[field] || '-';

  return (
    <Card title={t('profile.title')} className="portal-profile-card">
      <Descriptions column={{ xs: 1, sm: 2 }} bordered items={[
        { key: 'avatar', label: t('profile.avatar'), children: <Avatar size={56} src={user?.avatar_url}>{user?.display_name?.slice(0, 1)}</Avatar> },
        { key: 'display-name', label: t('profile.displayName'), children: value('display_name') },
        { key: 'english-name', label: t('profile.englishName'), children: value('english_name') },
        { key: 'username', label: t('users.username'), children: value('username') },
        { key: 'employee-number', label: t('profile.employeeNo'), children: value('employee_no') },
        { key: 'job-title', label: t('profile.jobTitle'), children: value('job_title') },
        { key: 'email', label: t('users.email'), children: value('email') },
        { key: 'phone', label: t('profile.phone'), children: value('phone') },
        { key: 'status', label: t('profile.status'), children: value('lifecycle_status') },
        { key: 'type', label: t('profile.userType'), children: value('user_type') },
        { key: 'source', label: t('profile.primarySource'), children: value('primary_source_name') },
        { key: 'locale', label: t('profile.locale'), children: value('locale') },
      ]} />
    </Card>
  );
}
