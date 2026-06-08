export const getPostLoginRedirectPath = (user) => {
  if (user?.role === 'admin') {
    const firstLoginDone = localStorage.getItem('admin_first_login_done')
    return firstLoginDone ? '/dashboard' : '/admin/config-wizard'
  }

  return '/agents'
}
