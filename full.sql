\set ON_ERROR_STOP on
drop DATABASE IF EXISTS ai_task;
create DATABASE ai_task;
\c ai_task;

-- =============================================
-- 用户画像表
-- =============================================
CREATE TABLE IF NOT EXISTS user_profile (
    id BIGSERIAL PRIMARY KEY,                                    -- 主键ID
    user_id VARCHAR(64) NOT NULL,                                -- 用户ID
    key VARCHAR(255) NOT NULL,                                   -- 属性键名
    value TEXT NOT NULL,                                         -- 属性值
    confidence REAL DEFAULT 0,                                   -- 置信度(0-1)
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,              -- 更新时间
    source_msg_id BIGINT,                                        -- 来源消息ID
    meta JSONB DEFAULT '{}',                                     -- 元数据(JSON格式)
    UNIQUE(user_id, key)
);

COMMENT ON TABLE user_profile IS '用户画像表，存储用户偏好和配置信息';
COMMENT ON COLUMN user_profile.id IS '主键ID，自增';
COMMENT ON COLUMN user_profile.user_id IS '用户唯一标识';
COMMENT ON COLUMN user_profile.key IS '属性键名，如preference_language、timezone等';
COMMENT ON COLUMN user_profile.value IS '属性值';
COMMENT ON COLUMN user_profile.confidence IS '置信度，范围0-1，表示该属性的可信程度';
COMMENT ON COLUMN user_profile.updated_at IS '最后更新时间';
COMMENT ON COLUMN user_profile.source_msg_id IS '该属性提取自哪条消息';
COMMENT ON COLUMN user_profile.meta IS '扩展元数据，JSON格式';

CREATE INDEX idx_user_profile_user_id ON user_profile(user_id);
CREATE INDEX idx_user_profile_key ON user_profile(key);

-- =============================================
-- 任务表
-- 存储任务的基本信息和状态，对应 Manus 的 task_plan.md
-- =============================================
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(64) PRIMARY KEY,                                  -- 任务ID(UUID)
    user_id VARCHAR(64) NOT NULL,                                -- 用户ID
    session_id VARCHAR(64) NOT NULL,                             -- 会话ID
    goal TEXT NOT NULL,                                          -- 任务目标
    current_phase VARCHAR(64),                                   -- 当前阶段ID
    phases_json TEXT,                                            -- 阶段列表(JSON数组)
    questions_json TEXT,                                         -- 关键问题(JSON数组)
    decisions_json TEXT,                                         -- 决策记录(JSON数组)
    errors_json TEXT,                                            -- 错误记录(JSON数组)
    status VARCHAR(32) DEFAULT 'pending',                        -- 任务状态: pending/in_progress/completed/failed/cancelled
    tool_call_count INT DEFAULT 0,                               -- 工具调用计数
    needs_reread BOOLEAN DEFAULT FALSE,                          -- 是否需要重读计划
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,              -- 创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,              -- 更新时间
    completed_at TIMESTAMP                                       -- 完成时间
);

COMMENT ON TABLE tasks IS '任务表，存储任务规划和执行状态，实现Manus的task_plan.md功能';
COMMENT ON COLUMN tasks.id IS '任务唯一标识，UUID格式';
COMMENT ON COLUMN tasks.user_id IS '创建任务的用户ID';
COMMENT ON COLUMN tasks.session_id IS '任务所属的会话ID';
COMMENT ON COLUMN tasks.goal IS '任务目标描述';
COMMENT ON COLUMN tasks.current_phase IS '当前正在执行的阶段ID';
COMMENT ON COLUMN tasks.phases_json IS '任务阶段列表，JSON格式，包含阶段ID、名称、描述、状态、步骤等';
COMMENT ON COLUMN tasks.questions_json IS '关键问题列表，JSON数组格式';
COMMENT ON COLUMN tasks.decisions_json IS '决策记录列表，JSON格式，包含决策内容、理由、时间戳等';
COMMENT ON COLUMN tasks.errors_json IS '错误记录列表，JSON格式，包含错误信息、尝试次数、解决方案等';
COMMENT ON COLUMN tasks.status IS '任务状态：pending-待处理、in_progress-进行中、completed-已完成、failed-失败、cancelled-已取消';
COMMENT ON COLUMN tasks.tool_call_count IS '工具调用计数，用于判断何时需要重读计划（Manus的10次规则）';
COMMENT ON COLUMN tasks.needs_reread IS '是否需要重读计划标记';
COMMENT ON COLUMN tasks.created_at IS '任务创建时间';
COMMENT ON COLUMN tasks.updated_at IS '任务最后更新时间';
COMMENT ON COLUMN tasks.completed_at IS '任务完成时间';

CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_session_id ON tasks(session_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);

-- =============================================
-- 任务发现表
-- 存储任务执行过程中的发现和资源，对应 Manus 的 findings.md
-- =============================================
CREATE TABLE IF NOT EXISTS task_findings (
    id BIGSERIAL PRIMARY KEY,                                    -- 主键ID
    task_id VARCHAR(64) NOT NULL,                                -- 关联的任务ID
    requirements_json TEXT,                                      -- 需求列表(JSON数组)
    findings_json TEXT,                                          -- 发现列表(JSON数组)
    resources_json TEXT,                                         -- 资源列表(JSON数组)
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP               -- 更新时间
);

COMMENT ON TABLE task_findings IS '任务发现表，存储任务执行过程中的研究发现和资源，实现Manus的findings.md功能';
COMMENT ON COLUMN task_findings.id IS '主键ID，自增';
COMMENT ON COLUMN task_findings.task_id IS '关联的任务ID';
COMMENT ON COLUMN task_findings.requirements_json IS '需求列表，JSON数组格式';
COMMENT ON COLUMN task_findings.findings_json IS '发现列表，JSON格式，包含类别、内容、来源、时间戳等';
COMMENT ON COLUMN task_findings.resources_json IS '资源列表，JSON数组格式，存储相关文件路径、URL等';
COMMENT ON COLUMN task_findings.updated_at IS '最后更新时间';

CREATE INDEX idx_task_findings_task_id ON task_findings(task_id);

-- =============================================
-- 任务进度表
-- 存储任务执行的详细进度，对应 Manus 的 progress.md
-- =============================================
CREATE TABLE IF NOT EXISTS task_progress (
    id BIGSERIAL PRIMARY KEY,                                    -- 主键ID
    task_id VARCHAR(64) NOT NULL,                                -- 关联的任务ID
    session_date VARCHAR(16),                                    -- 会话日期(YYYY-MM-DD格式)
    entries_json TEXT,                                           -- 进度条目(JSON数组)
    test_results_json TEXT,                                      -- 测试结果(JSON数组)
    error_log_json TEXT,                                         -- 错误日志(JSON数组)
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP               -- 更新时间
);

COMMENT ON TABLE task_progress IS '任务进度表，存储任务执行的详细进度和测试结果，实现Manus的progress.md功能';
COMMENT ON COLUMN task_progress.id IS '主键ID，自增';
COMMENT ON COLUMN task_progress.task_id IS '关联的任务ID';
COMMENT ON COLUMN task_progress.session_date IS '会话日期，格式YYYY-MM-DD';
COMMENT ON COLUMN task_progress.entries_json IS '进度条目列表，JSON格式，包含阶段ID、动作、文件列表、时间戳等';
COMMENT ON COLUMN task_progress.test_results_json IS '测试结果列表，JSON格式，包含测试名称、输入、预期、实际、状态等';
COMMENT ON COLUMN task_progress.error_log_json IS '错误日志列表，JSON格式，与tasks.errors_json结构相同';
COMMENT ON COLUMN task_progress.updated_at IS '最后更新时间';

CREATE INDEX idx_task_progress_task_id ON task_progress(task_id);
