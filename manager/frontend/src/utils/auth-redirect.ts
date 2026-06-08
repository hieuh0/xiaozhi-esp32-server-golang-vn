export function getPostLoginRedirectPath(role: string): string {
  if (role === 'admin') {
    return localStorage.getItem('admin_first_login_done') ? '/dashboard' : '/admin/config-wizard'
  }
  return '/agents'
}
