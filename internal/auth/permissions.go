package auth

import "encoding/json"

const (
	PermViewInbox           = "view_inbox"
	PermViewAccounts        = "view_accounts"
	PermViewWebhooks        = "view_webhooks"
	PermViewSettings        = "view_settings"
	PermViewUsers           = "view_users"
	PermActionReply         = "action_reply"
	PermActionDelivery      = "action_delivery"
	PermActionAccountsWrite = "action_accounts_write"
	PermActionDataPurge     = "action_data_purge"
	PermActionUsersManage   = "action_users_manage"
)

func AllPermissions() []string {
	return []string{
		PermViewInbox,
		PermViewAccounts,
		PermViewWebhooks,
		PermViewSettings,
		PermViewUsers,
		PermActionReply,
		PermActionDelivery,
		PermActionAccountsWrite,
		PermActionDataPurge,
		PermActionUsersManage,
	}
}

func HasPermission(isAdmin bool, permissions []string, perm string) bool {
	if isAdmin {
		return true
	}
	for _, p := range permissions {
		if p == perm {
			return true
		}
	}
	return false
}

func ParsePermissions(raw []byte) []string {
	var perms []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &perms)
	}
	return perms
}

func PermissionsJSON(perms []string) json.RawMessage {
	if perms == nil {
		perms = []string{}
	}
	b, _ := json.Marshal(perms)
	return b
}
