package data

import (
	"fmt"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var managedLabel = map[string]string{"harvesterhci.io/managed": "true"}

const (
	harvesterRoleNamePrefix = "harvester-"
)

type roleBuilder struct {
	previous          *roleBuilder
	next              *roleBuilder
	name              string
	displayName       string
	context           string
	builtin           bool
	external          bool
	hidden            bool
	administrative    bool
	roleTemplateNames []string
	rules             []*ruleBuilder
}

func (rb *roleBuilder) String() string {
	return fmt.Sprintf("%s (%s): %s", rb.displayName, rb.name, rb.rules)
}

func newRoleBuilder() *roleBuilder {
	return &roleBuilder{
		builtin: true,
	}
}

func (rb *roleBuilder) addRoleTemplate(displayName, name, context string, external, hidden, administrative bool) *roleBuilder {
	r := rb.addRole(displayName, name)
	r.context = context
	r.external = external
	r.hidden = hidden
	r.administrative = administrative
	return r
}

func (rb *roleBuilder) addRole(displayName, name string) *roleBuilder {
	if rb.name == "" {
		rb.name = name
		rb.displayName = displayName
		return rb
	}

	if rb.next != nil {
		return rb.next.addRole(displayName, name)
	}
	rb.next = newRoleBuilder()
	rb.next.name = name
	rb.next.displayName = displayName
	rb.next.previous = rb

	return rb.next
}

func (rb *roleBuilder) addRule() *ruleBuilder {
	r := &ruleBuilder{
		rb: rb,
	}
	rb.rules = append(rb.rules, r)
	return r
}

func (rb *roleBuilder) setRoleTemplateNames(names ...string) *roleBuilder {
	rb.roleTemplateNames = names
	return rb
}

func (rb *roleBuilder) first() *roleBuilder {
	if rb.previous == nil {
		return rb
	}
	return rb.previous.first()
}

func (rb *roleBuilder) policyRules() []rbacv1.PolicyRule {
	if len(rb.rules) == 0 {
		return nil
	}
	prs := make([]rbacv1.PolicyRule, len(rb.rules))
	for i, r := range rb.rules {
		prs[i] = r.toPolicyRule()
	}
	return prs
}

type ruleBuilder struct {
	rb             *roleBuilder
	_verbs         []string
	_resources     []string
	_resourceNames []string
	_apiGroups     []string
	_urls          []string
}

func (r *ruleBuilder) String() string {
	return fmt.Sprintf("apigroups: %v, resource: %v, resourceNames: %v, nonResourceURLs %v, verbs: %v", r._apiGroups, r._resources, r._resourceNames, r._urls, r._verbs)
}

func (r *ruleBuilder) verbs(v ...string) *ruleBuilder {
	r._verbs = append(r._verbs, v...)
	return r
}

func (r *ruleBuilder) resources(rc ...string) *ruleBuilder {
	r._resources = append(r._resources, rc...)
	return r
}

func (r *ruleBuilder) resourceNames(rn ...string) *ruleBuilder {
	r._resourceNames = append(r._resourceNames, rn...)
	return r
}

func (r *ruleBuilder) apiGroups(a ...string) *ruleBuilder {
	r._apiGroups = append(r._apiGroups, a...)
	return r
}

func (r *ruleBuilder) nonResourceURLs(u ...string) *ruleBuilder {
	r._urls = append(r._urls, u...)
	return r
}

func (r *ruleBuilder) addRoleTemplate(displayName, name, context string, external, hidden, administrative bool) *roleBuilder {
	return r.rb.addRoleTemplate(displayName, name, context, external, hidden, administrative)
}

func (r *ruleBuilder) addRole(displayName, name string) *roleBuilder {
	return r.rb.addRole(displayName, name)
}

func (r *ruleBuilder) addRule() *ruleBuilder {
	return r.rb.addRule()
}

func (r *ruleBuilder) setRoleTemplateNames(names ...string) *roleBuilder {
	return r.rb.setRoleTemplateNames(names...)
}

func (r *ruleBuilder) toPolicyRule() rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		APIGroups:       r._apiGroups,
		Resources:       r._resources,
		ResourceNames:   r._resourceNames,
		NonResourceURLs: r._urls,
		Verbs:           r._verbs,
	}
}

func (rb *roleBuilder) buildGlobalRoles() []runtime.Object {
	var result []runtime.Object
	for role := rb.first(); role != nil; role = role.next {
		result = append(result, &v3.GlobalRole{
			ObjectMeta: v1.ObjectMeta{
				Name:   harvesterRoleNamePrefix + role.name,
				Labels: managedLabel,
			},
			DisplayName: role.displayName,
			Rules:       role.policyRules(),
			Builtin:     role.builtin,
		})
	}

	return result
}

func (rb *roleBuilder) buildRoleTemplates() []runtime.Object {
	var result []runtime.Object
	for roleTemplate := rb.first(); roleTemplate != nil; roleTemplate = roleTemplate.next {
		result = append(result, &v3.RoleTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name:   harvesterRoleNamePrefix + roleTemplate.name,
				Labels: managedLabel,
			},
			DisplayName:       roleTemplate.displayName,
			Builtin:           roleTemplate.builtin,
			External:          roleTemplate.external,
			Hidden:            roleTemplate.hidden,
			Context:           roleTemplate.context,
			Rules:             roleTemplate.policyRules(),
			RoleTemplateNames: roleTemplate.roleTemplateNames,
			Administrative:    roleTemplate.administrative,
		})
	}

	return result
}
