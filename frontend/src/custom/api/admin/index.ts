import { adminAPI as upstreamAdminAPI } from '@/api/admin'
import accountsAPI from '@/custom/api/admin/accounts'
import channelMonitorAPI from '@/custom/api/admin/channelMonitor'
import groupsAPI from '@/custom/api/admin/groups'
import settingsAPI from '@/custom/api/admin/settings'
import usersAPI from '@/custom/api/admin/users'
import upstreamsAPI from '@/custom/api/admin/upstreams'

export const adminAPI = {
  ...upstreamAdminAPI,
  accounts: accountsAPI,
  channelMonitor: channelMonitorAPI,
  groups: groupsAPI,
  settings: settingsAPI,
  users: usersAPI,
  upstreams: upstreamsAPI,
}

export { accountsAPI, channelMonitorAPI, groupsAPI, settingsAPI, usersAPI, upstreamsAPI }
export type { BalanceHistoryItem } from '@/custom/api/admin/users'

export default adminAPI
