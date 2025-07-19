package repository

type ChatRobot struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// 注意：以下方法返回原始错误，由路由层处理响应

func (r *Repository) CreateRobot(robot *ChatRobot) error {
	return r.db.Create(robot).Error
}
func (r *Repository) DeleteRobot(id int) error {
	return r.db.Delete(&ChatRobot{}, id).Error
}
func (r *Repository) UpdateRobot(robot *ChatRobot) error {
	return r.db.Save(robot).Error
}
func (r *Repository) GetAllRobots() ([]ChatRobot, error) {
	var robots []ChatRobot
	err := r.db.Find(&robots).Error
	return robots, err
}
