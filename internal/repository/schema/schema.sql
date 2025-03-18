CREATE TABLE tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,      -- 任务 ID（自增）
    task_id CHAR(36) NOT NULL UNIQUE,       -- 任务唯一标识（UUID）
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 任务状态: pending, running, success, failed
    result JSON DEFAULT NULL,               -- 任务结果（JSON 类型）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 任务创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- 任务更新时间
);