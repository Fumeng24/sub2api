-- Migration: 160_drop_channel_monitor_account_id
-- 清理 migration 158 在 channel_monitors 上加的 account_id 孤儿列。
--
-- 背景：158 曾为"账号级监控复用 channel-monitor"加了 account_id（列 + FK + 索引）。
-- 该方案已废弃——账号监控改为完全独立的一套（account_monitors 表，见 159），
-- channel-monitor 回退为纯站点级监控。158 的 SQL 文件已删除，但生产 DB 已执行过它，
-- 残留列/约束/索引是孤儿（代码与 ent schema 均不再引用）。
--
-- 安全性：该列从未真正使用（线上 account_id 全为 NULL），DROP 无数据损失。
-- 全部 IF EXISTS，可重复执行。

ALTER TABLE channel_monitors DROP CONSTRAINT IF EXISTS fk_channel_monitors_account_id;
DROP INDEX IF EXISTS idx_channel_monitors_account_id;
ALTER TABLE channel_monitors DROP COLUMN IF EXISTS account_id;

-- 158 文件已从代码库删除；清掉它在 schema_migrations 中的记录，
-- 让"已应用迁移列表"与磁盘上的迁移文件保持一致。
DELETE FROM schema_migrations WHERE filename = '158_add_channel_monitor_account_id.sql';
