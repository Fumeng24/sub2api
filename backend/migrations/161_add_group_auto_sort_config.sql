-- Migration: 161_add_group_auto_sort_config
-- 分组级「持续自动排序」配置。后端定时任务(GroupAutoSortService, 每分钟)按此配置
-- 周期性重排该分组成员账号的 priority：
--   - basis = 'rate'         按最终倍率升序（低倍率优先）
--   - basis = 'availability' 按近 1 小时可用率降序（高可用优先，数据源 account_monitor_checks）
-- 仅 enabled = true 的分组参与；空配置 {} 等价于关闭。
--
-- 存储：groups.auto_sort_config (jsonb)，结构 {"enabled":bool,"basis":"rate"|"availability"}。
-- 与 ent schema field.JSON("auto_sort_config", domain.GroupAutoSortConfig{}) 对应。

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS auto_sort_config JSONB NOT NULL DEFAULT '{}'::jsonb;
