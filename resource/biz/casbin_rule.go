package biz

import (
	"context"

	_string "github.com/liujitcn/go-utils/string"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// CasbinRuleCase 提供 Casbin 权限规则重建能力。
type CasbinRuleCase struct {
	permissionStore data.PermissionStore
	baseAPICase     *BaseAPICase
	authzEngine     engine.Engine
}

// NewCasbinRuleCase 创建权限规则业务实例。
func NewCasbinRuleCase(
	permissionStore data.PermissionStore,
	baseAPICase *BaseAPICase,
	authzEngine engine.Engine,
) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		permissionStore: permissionStore,
		baseAPICase:     baseAPICase,
		authzEngine:     authzEngine,
	}, nil
}

// RebuildPolicy 按角色、菜单和 API 全量重建 Casbin 规则与内存策略。
func (c *CasbinRuleCase) RebuildPolicy(ctx context.Context) error {
	err := c.RebuildPolicyData(ctx)
	if err != nil {
		return err
	}
	return c.RefreshPolicy(ctx)
}

// RebuildPolicyData 按角色、菜单和 API 重建数据库中的 Casbin 规则快照。
func (c *CasbinRuleCase) RebuildPolicyData(ctx context.Context) error {
	baseRoleList, err := c.permissionStore.ListRoles(ctx)
	if err != nil {
		return err
	}
	var baseTenantList []data.TenantRecord
	baseTenantList, err = c.permissionStore.ListTenants(ctx)
	if err != nil {
		return err
	}

	// 仅查询角色实际关联的菜单，减少无关菜单参与规则构建。
	menuIDSet := make(map[int64]struct{})
	for _, item := range baseRoleList {
		for _, menuID := range _string.ConvertJsonStringToInt64Array(item.Menus) {
			menuIDSet[menuID] = struct{}{}
		}
	}
	menuIDs := make([]int64, 0, len(menuIDSet))
	for menuID := range menuIDSet {
		menuIDs = append(menuIDs, menuID)
	}
	var baseMenuList []data.MenuRecord
	baseMenuList, err = c.permissionStore.ListMenusByIDs(ctx, menuIDs)
	if err != nil {
		return err
	}
	var baseAPIList []*data.APIPolicyRecord
	baseAPIList, err = c.baseAPICase.FindAll(ctx)
	if err != nil {
		return err
	}

	// 根据读取到的角色、租户、菜单和 API 数据构造完整规则快照。
	casbinRuleList := buildCasbinRuleList(baseRoleList, baseTenantList, baseMenuList, baseAPIList)
	err = c.permissionStore.ReplacePolicies(ctx, casbinRuleList)
	if err != nil {
		return err
	}
	return nil
}

// RefreshPolicy 根据数据库规则刷新内存权限策略。
func (c *CasbinRuleCase) RefreshPolicy(ctx context.Context) error {
	policyRule := make([]casbin.PolicyRule, 0)
	// 为默认租户的超级角色授予全部 API 权限。
	baseAPIList, err := c.baseAPICase.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, item := range baseAPIList {
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: "p",
			V0:    gorm.DefaultTenantCode,
			V1:    _const.BASE_ROLE_CODE_SUPER,
			V2:    item.Operation,
			V3:    item.Method,
			V4:    "*",
		})
	}
	// 读取数据库中的租户角色权限规则。
	var casbinRuleList []data.PolicyRecord
	casbinRuleList, err = c.permissionStore.ListPolicies(ctx)
	if err != nil {
		return err
	}
	for _, item := range casbinRuleList {
		// 旧版本策略缺少租户或项目占位字段时会被 Casbin 识别为 4 段规则，启动阶段直接跳过等待角色权限重建修复。
		if item.Ptype == "" || item.V0 == "" || item.V1 == "" || item.V2 == "" || item.V3 == "" || item.V4 == "" {
			continue
		}
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
		})
	}
	policyMap := make(engine.PolicyMap)
	policyMap["policies"] = policyRule
	roleMap := make(engine.RoleMap)
	writer, ok := c.authzEngine.(engine.PolicyWriter)
	if !ok {
		return errorsx.Internal("鉴权引擎不支持策略写入")
	}
	return writer.SetPolicies(ctx, policyMap, roleMap)
}

// buildCasbinRuleList 根据角色菜单、租户和接口关联构造去重后的 Casbin 策略。
func buildCasbinRuleList(baseRoleList []data.RoleRecord, baseTenantList []data.TenantRecord, baseMenuList []data.MenuRecord, baseAPIList []*data.APIPolicyRecord) []data.PolicyRecord {
	tenantCodeByID := make(map[int64]string, len(baseTenantList))
	for _, item := range baseTenantList {
		tenantCodeByID[item.ID] = item.Code
	}
	menuOperationsByID := make(map[int64][]string, len(baseMenuList))
	for _, item := range baseMenuList {
		menuOperationsByID[item.ID] = _string.ConvertJsonStringToStringArray(item.API)
	}
	apiByOperation := make(map[string]*data.APIPolicyRecord, len(baseAPIList))
	for _, item := range baseAPIList {
		if _, ok := apiByOperation[item.Operation]; !ok {
			apiByOperation[item.Operation] = item
		}
	}

	rules := make([]data.PolicyRecord, 0)
	ruleSet := make(map[string]struct{})
	for _, baseRole := range baseRoleList {
		tenantCode, ok := tenantCodeByID[baseRole.TenantID]
		// 角色所属租户不存在时，不生成无效策略。
		if !ok {
			continue
		}
		for _, menuID := range _string.ConvertJsonStringToInt64Array(baseRole.Menus) {
			for _, operation := range menuOperationsByID[menuID] {
				baseAPI, ok := apiByOperation[operation]
				// 菜单关联的接口已失效时，不生成无效策略。
				if !ok {
					continue
				}
				ruleKey := tenantCode + "\x00" + baseRole.Code + "\x00" + baseAPI.Operation + "\x00" + baseAPI.Method
				if _, ok = ruleSet[ruleKey]; ok {
					continue
				}
				ruleSet[ruleKey] = struct{}{}
				rules = append(rules, data.PolicyRecord{
					Ptype: "p",
					V0:    tenantCode,
					V1:    baseRole.Code,
					V2:    baseAPI.Operation,
					V3:    baseAPI.Method,
					V4:    "*",
				})
			}
		}
	}
	return rules
}
