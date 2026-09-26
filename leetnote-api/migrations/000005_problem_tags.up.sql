-- =============================================================
-- 000005_problem_tags · 题目 ↔ 标签（多对多）
--
-- 为什么需要：灵神的题单是【按专题组织】的（相向双指针、滑动窗口、二分…），
-- 导入时必须把专题信息一起存下来，否则就丢掉了题单最有价值的部分。
--
-- 为什么不直接在 problems 上加一个 topic 字段：
--   一道题可能同时属于多个专题（比如「二分答案」和「最小化最大值」），
--   单列存不下；而且标签体系将来还要复用到别的维度。
--
-- 结构与 note_tags 对称，便于复用同一套「批量查、批量关联」的代码。
-- =============================================================

CREATE TABLE problem_tags (
    problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags(id)     ON DELETE CASCADE,
    PRIMARY KEY (problem_id, tag_id)
);

-- 反向查询用：按标签筛题目
CREATE INDEX idx_problem_tags_tag ON problem_tags (tag_id);
