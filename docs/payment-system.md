# Enterprise Payment System Guide

[简体中文](#中文版本) | [English](#english-version)

---

<a name="中文版本"></a>
## 中文版本

### 概述

Open Station 提供完整的企业支付系统，支持个人和组织双模式，包含双层配额控制、后付费信用额度管理、多支付渠道集成等功能。

---

### 双用户模式

#### 个人模式 (Individual Mode)

适用场景: 公共租户用户，完全独立的配额管理

```
┌─────────────────────────────────────────────────────────────────┐
│                  公共租户 (Public Tenant)                        │
│                                                                 │
│  每个用户完全独立，拥有自己的:                                    │
│  ┌──────────────────────┐  ┌──────────────────────┐            │
│  │ UserQuota (用户A)     │  │ UserQuota (用户B)     │            │
│  │                      │  │                      │            │
│  │ ├─ Balance: $50      │  │ ├─ Balance: $100     │            │
│  │ ├─ TokenQuota: 100K  │  │ ├─ TokenQuota: 50K   │            │
│  │ ├─ TokensUsed: 80K   │  │ ├─ TokensUsed: 30K   │            │
│  │ └─ Status: active    │  │ └─ Status: active    │            │
│  └──────────────────────┘  └──────────────────────┘            │
│                                                                 │
│  计费模式: 预充值 + 套餐 (无后付费)                               │
│  停服条件: 余额为0且套餐Token用完 → 停服                         │
└─────────────────────────────────────────────────────────────────┘
```

#### 组织模式 (Organization Mode)

适用场景: 企业租户，共享租户资源 + 成员独立控制

```
┌─────────────────────────────────────────────────────────────────┐
│                  企业租户 (Tenant)                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────┐       │
│  │ 租户共享资源                                          │       │
│  │ ├─ Balance: $10,000 (租户总余额)                      │       │
│  │ ├─ TokenLimit: 1,000,000 (租户总Token限制)            │       │
│  │ ├─ CreditLimit: $20,000 (后付费信用额度，审核后开通)   │       │
│  │ ├─ CreditStatus: approved (审核状态)                  │       │
│  │ └─────────────────────────────────────────────────────┘       │
│                              ↓                                   │
│  ┌────────────────────┐  ┌────────────────────┐                  │
│  │ MemberQuota (A)    │  │ MemberQuota (B)    │                  │
│  │ (管理员)           │  │ (开发成员)         │                  │
│  │                    │  │                    │                  │
│  │ ├─ TokenQuotaLimit │  │ ├─ TokenQuotaLimit │                  │
│  │ │   : 500K         │  │ │   : 200K         │                  │
│  │ ├─ CostLimit: $2K  │  │ ├─ CostLimit: $500 │                  │
│  │ └─ Status: active  │  │ └─ Status: active  │                  │
│  └────────────────────┘  └────────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
```

---

### 统一扣减优先级

无论个人还是组织，费用扣除优先级一致:

```
优先级 1: 套餐Token额度
         ↓ 用完后
优先级 2: 预充值余额
         ↓ 余额为0时
优先级 3: 后付费信用额度 (个人无此项，仅企业审核通过后可用)
         ↓ 超限后
停服
```

**停服条件:**
- 个人: 余额为0且无套餐Token → 停服
- 企业: 余额为0且无信用额度(或信用未审核通过) → 停服
- 成员超限: 成员Token配额超限或费用限额超限 → 成员停服

---

### 后付费申请流程

#### 1. 企业申请

```bash
POST /tenant/credit-application
Authorization: Bearer sk-tenant-admin-key
Content-Type: application/json

{
  "requested_limit": 20000,
  "reason": "月度结算需求",
  "settlement_cycle": "monthly",
  "settlement_day": 25
}
```

#### 2. 平台管理员审核

```bash
POST /platform/credit-applications/:id/review
Authorization: Bearer sk-platform-admin-key
Content-Type: application/json

{
  "status": "approved",
  "approved_limit": 15000,
  "review_notes": "通过，额度调整为15000"
}
```

#### 3. 审核结果

- **通过**: Tenant.CreditLimit = ApprovedLimit, Tenant.CreditStatus = approved
- **拒绝**: Tenant.CreditStatus = rejected, Tenant.CreditRejectReason = 原因

---

### 结算流程

#### 结算周期

| 类型 | 说明 |
|------|------|
| **月结** | 每月 SettlementDay 日触发结算 |
| **周结** | 每 SettlementDay (周几) 触发结算 |
| **阈值触发** | CreditUsed >= ThresholdAmount 时触发 |
| **自定义** | 手动触发或自定义规则 |

#### 结算流程

```
结算触发 → 生成账单 → 发送账单通知 → 企业支付账单 → 支付成功 → 重置信用使用
```

---

### 成员配额管理

#### 创建成员配额

```bash
POST /admin/member-quotas
Authorization: Bearer sk-tenant-admin-key
Content-Type: application/json

{
  "user_id": "xxx",
  "token_quota_limit": 200000,
  "cost_limit": 500,
  "cost_limit_type": "monthly",
  "max_api_keys": 5
}
```

#### 设置成员Token配额

```bash
PUT /admin/member-quotas/:id/token-limit
Content-Type: application/json

{
  "limit": 500000
}
```

#### 设置成员费用限额

```bash
PUT /admin/member-quotas/:id/cost-limit
Content-Type: application/json

{
  "limit": 1000,
  "limit_type": "monthly"
}
```

---

### 支付渠道

| 渠道 | 适用地区 | 支付方式 | 特点 |
|------|----------|----------|------|
| **支付宝** | 国内 | 扫码、网页、APP | 实时到账 |
| **微信支付** | 国内 | 扫码、网页、APP | 实时到账 |
| **Stripe** | 国际 | 信用卡、网页 | 支持多币种 |
| **PayPal** | 国际 | 账户余额、信用卡 | 全球覆盖 |
| **银行转账** | 企业 | 线下转账 | 大额支付 |

---

### API 端点

#### 个人用户路由

```bash
GET  /user/quota             # 查看个人配额
GET  /user/balance           # 查看余额
GET  /user/usage             # 使用统计
GET  /user/recharge-history  # 充值记录
```

#### 租户管理员路由

```bash
# 后付费申请
POST /tenant/credit-application
GET  /tenant/credit-application
PUT  /tenant/credit-application
DELETE /tenant/credit-application

# 成员配额管理
GET  /admin/member-quotas
POST /admin/member-quotas
PUT  /admin/member-quotas/:id
DELETE /admin/member-quotas/:id
PUT  /admin/member-quotas/:id/token-limit
PUT  /admin/member-quotas/:id/cost-limit
GET  /admin/member-quotas/:id/usage
POST /admin/member-quotas/:id/reset
```

#### 平台管理员路由

```bash
# 后付费审核
GET  /platform/credit-applications
GET  /platform/credit-applications/:id
POST /platform/credit-applications/:id/review
PUT  /platform/tenants/:id/credit

# 成员配额查看
GET  /platform/member-quotas
```

#### 支付路由

```bash
POST /payment/orders
GET  /payment/orders/:id
POST /payment/orders/:id/cancel

POST /payment/callback/alipay
POST /payment/callback/wechat
POST /payment/callback/stripe
POST /payment/callback/paypal
```

---

### 错误码

| 错误码 | 说明 |
|--------|------|
| `QUOTA_014` | Token配额超限 |
| `QUOTA_015` | 租户Token配额超限 |
| `QUOTA_017` | 成员Token配额超限 |
| `QUOTA_018` | 成员费用限额超限 |
| `QUOTA_021` | 信用额度超限 |
| `QUOTA_022` | 信用额度未审核通过 |

---

<a name="english-version"></a>
## English Version

### Overview

Open Station provides a complete enterprise payment system supporting dual user modes (individual and organization), including dual quota control, postpaid credit management, and multi-channel payment integration.

---

### Dual User Modes

#### Individual Mode

Use case: Public tenant users with completely independent quota management

- **Quota Source**: UserQuota (independent)
- **Billing Mode**: Prepaid + Subscription (no postpaid)
- **Service Stop**: Balance = 0 and subscription tokens exhausted → Stop service

#### Organization Mode

Use case: Enterprise tenants with shared resources + member control

- **Quota Source**: Tenant shared + MemberQuota control
- **Billing Mode**: Prepaid + Subscription + Postpaid (requires approval)
- **Service Stop**: Balance = 0 and no approved credit → Stop service

---

### Unified Deduction Priority

```
Priority 1: Subscription Token Quota (first deduction)
            ↓ after exhausted
Priority 2: Prepaid Balance
            ↓ when balance = 0
Priority 3: Postpaid Credit Limit (individuals don't have this)
            ↓ over limit
Stop Service
```

---

### Postpaid Application Workflow

#### 1. Enterprise Application

```bash
POST /tenant/credit-application
Authorization: Bearer sk-tenant-admin-key

{
  "requested_limit": 20000,
  "reason": "Monthly settlement requirement",
  "settlement_cycle": "monthly",
  "settlement_day": 25
}
```

#### 2. Platform Admin Review

```bash
POST /platform/credit-applications/:id/review
Authorization: Bearer sk-platform-admin-key

{
  "status": "approved",
  "approved_limit": 15000,
  "review_notes": "Approved, limit adjusted to 15000"
}
```

---

### Settlement Cycles

| Type | Description |
|------|-------------|
| **Monthly** | Triggered on SettlementDay each month |
| **Weekly** | Triggered on SettlementDay (day of week) |
| **Threshold** | Triggered when CreditUsed >= ThresholdAmount |
| **Custom** | Manual trigger or custom rules |

---

### Member Quota Management

#### Create Member Quota

```bash
POST /admin/member-quotas

{
  "user_id": "xxx",
  "token_quota_limit": 200000,
  "cost_limit": 500,
  "cost_limit_type": "monthly"
}
```

#### Set Token Quota Limit

```bash
PUT /admin/member-quotas/:id/token-limit

{
  "limit": 500000
}
```

#### Set Cost Limit

```bash
PUT /admin/member-quotas/:id/cost-limit

{
  "limit": 1000,
  "limit_type": "monthly"
}
```

---

### Payment Channels

| Channel | Region | Payment Methods |
|---------|--------|-----------------|
| **Alipay** | China | QR code, Web, APP |
| **WeChat Pay** | China | QR code, Web, APP |
| **Stripe** | International | Credit card, Web |
| **PayPal** | International | Account balance, Credit card |
| **Bank Transfer** | Enterprise | Offline transfer |

---

### Error Codes

| Code | Description |
|------|-------------|
| `QUOTA_014` | Token quota exceeded |
| `QUOTA_015` | Tenant token quota exceeded |
| `QUOTA_017` | Member token quota exceeded |
| `QUOTA_018` | Member cost limit exceeded |
| `QUOTA_021` | Credit limit exceeded |
| `QUOTA_022` | Credit not approved |

---

**Made with ❤️ by the Open Station Team**