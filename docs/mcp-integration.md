# MCP Service Integration Guide

This guide explains how to use the MCP (Model Context Protocol) Service with Claude Code CLI.

## Overview

Open Station provides an MCP server that allows Claude Code CLI to access gateway management tools and resources directly.

## Features

### User Tools (6 tools)
- `check_balance` - Check current token balance
- `get_usage_summary` - Get usage summary for a time period
- `get_usage_details` - Get detailed usage records
- `get_billing_info` - Get billing and payment information
- `get_recharge_history` - Get recharge/payment history
- `get_my_api_keys` - List user's own API keys

### Manager Tools (9 tools)
- `list_all_api_keys` - List all API keys in system
- `create_api_key` - Create new API key for user
- `revoke_api_key` - Revoke an API key
- `update_api_key` - Update API key permissions
- `list_users` - List all users
- `get_user_detail` - Get detailed user information
- `adjust_balance` - Adjust tenant balance
- `get_tenant_summary` - Get tenant summary
- `list_tenants` - List all tenants

## Configuration

### Claude Code CLI Setup

Add MCP server configuration to `~/.claude/settings.json` or `.claude/settings.local.json`:

```json
{
  "mcpServers": {
    "open-station": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer sk-your-api-key"
      }
    }
  }
}
```

### API Key Requirements

- **User Tools**: Any valid API key with `chat` permission
- **Manager Tools**: API key with `admin` or `manage` permission

## Usage Examples

### Checking Balance

```
User: What's my current balance?
Claude: [Uses check_balance tool]
Result: Current Balance: $125.50
```

### Viewing Usage

```
User: Show my usage for this month
Claude: [Uses get_usage_summary tool]
Result: Usage Summary (2025-01-01 to 2025-01-31):
- Total Tokens: 15,420
- Total Cost: $3.24
```

### Creating API Key (Manager)

```
User: Create a new API key for user John with chat permission
Claude: [Uses create_api_key tool]
Result: {
  "id": "uuid",
  "raw_key": "sk-xxx...",
  "name": "John's Key",
  "permissions": ["chat"]
}
```

### Adjusting Balance (Manager)

```
User: Add $50 to tenant abc-123 balance for monthly subscription
Claude: [Uses adjust_balance tool]
Result: {
  "tenant_id": "abc-123",
  "adjustment": "50",
  "reason": "monthly subscription",
  "new_balance": "175.50"
}
```

## Resources

MCP also provides read-only resources:

| URI | Description | Access |
|-----|-------------|--------|
| `user://profile` | Current user profile | User |
| `user://balance` | User's tenant balance | User |
| `user://usage` | User's usage records | User |
| `tenant://list` | All tenants list | Manager |
| `tenant://{id}` | Tenant details | Manager |
| `apikey://list` | All API keys | Manager |

## Protocol Details

- **Endpoint**: `/mcp`
- **Protocol**: JSON-RPC 2.0 over HTTP
- **Session**: Initialize creates session, session ID in header
- **Session Timeout**: 30 minutes

## Error Handling

- Permission denied errors for unauthorized tool access
- Tool execution errors returned with details
- Session expiration requires re-initialization

## Security

- API key required for all operations
- Role-based access control (user vs manager)
- Session isolation per API key