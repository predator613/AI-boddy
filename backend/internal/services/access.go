package services

import "ai-buddy/backend/internal/models"

// Правила доступа. Одно место, чтобы помощник и API одинаково скрывали чужие данные.
//
//   стажер      – видит только себя;
//   наставник   – видит своих подопечных (intern_profiles.mentor_id = он), меняет их статусы;
//   руководитель – видит и меняет всех стажеров своей команды.
//   Шаблоны адаптации команды редактируют наставник и руководитель этой команды.

// CanViewIntern – можно ли смотреть путь стажера.
func CanViewIntern(viewer, intern *models.User) bool {
	if viewer == nil || intern == nil || intern.Role != models.RoleIntern {
		return false
	}
	switch viewer.Role {
	case models.RoleIntern:
		return viewer.ID == intern.ID
	case models.RoleMentor:
		return intern.MentorID != nil && *intern.MentorID == viewer.ID
	case models.RoleLead:
		return viewer.TeamID == intern.TeamID
	}
	return false
}

// CanManageIntern – можно ли менять статусы задач и назначать встречи стажеру.
func CanManageIntern(viewer, intern *models.User) bool {
	return viewer != nil && viewer.Role != models.RoleIntern && CanViewIntern(viewer, intern)
}

// CanEditTemplates – можно ли редактировать шаблон адаптации команды.
func CanEditTemplates(viewer *models.User, teamID string) bool {
	return viewer != nil &&
		(viewer.Role == models.RoleMentor || viewer.Role == models.RoleLead) &&
		viewer.TeamID == teamID
}
