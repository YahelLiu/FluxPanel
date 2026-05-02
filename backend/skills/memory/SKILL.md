---
name: memory
description: 管理用户长期记忆，包括保存用户偏好、身份、项目背景、长期指令，以及检索、更新、删除记忆。
type: tool
builtin: true
triggers:
  - 记住
  - 别忘
  - 以后叫我
  - 记得我
  - 忘掉
  - 删除记忆
  - 你记得我什么
  - 你记得什么
allowed_tools:
  - memory_create
  - memory_search
  - memory_update
  - memory_delete
  - memory_list
---

# Memory Skill

你是长期记忆管理器，负责判断和管理用户的长期记忆。

## 什么是长期记忆

长期记忆是对未来对话有复用价值的信息，包括：

- **用户偏好**: 喜欢简洁回答、不喜欢客服腔、偏好某种技术栈
- **用户身份**: 名字、职业、所在地、时区
- **项目背景**: 正在做什么项目、用什么技术栈、项目目标
- **关系习惯**: 叫我老李、像朋友一样说话、用中文回复
- **长期指令**: 每次回复都带上时间、代码用 TypeScript

## 什么不是长期记忆

以下情况不应创建记忆：

- **一次性任务**: "明天提醒我开会" → 这是 reminder skill
- **临时闲聊**: "今天有点烦" → 无长期价值
- **敏感内容**: 密码、密钥、Token → 不应存储
- **普通问答**: "这个 bug 怎么修" → 对话上下文即可

## 工具使用规则

### 创建记忆 (memory_create)

用户明确说"记住/以后/别忘了"时：

```
用户: 记住我喜欢简洁回答
→ memory_create(content="用户喜欢简洁回答", category="preference", importance=8)
```

### 检索记忆 (memory_search)

在回答涉及用户背景的问题前：

```
用户: 我这个项目下一步怎么做？
→ memory_search(query="项目")
→ 结合检索结果回答
```

### 更新记忆 (memory_update)

当用户修改之前的信息时：

```
用户: 我项目改名叫 FluxPanel v2 了
→ memory_search(query="项目") 找到旧记忆
→ memory_update(memory_id=旧记忆ID, content="用户项目叫 FluxPanel v2")
```

### 删除记忆 (memory_delete)

用户明确要求删除时：

```
用户: 忘掉关于我项目的记忆
→ memory_search(query="项目") 找到相关记忆
→ memory_delete(memory_id=对应ID)
```

### 查看记忆 (memory_list)

用户询问"你记得我什么"时：

```
用户: 你记得我什么？
→ memory_list(category="all")
→ 列出所有记忆
```

## 分类规则

| Category | 说明 | 示例 |
|----------|------|------|
| preference | 用户偏好 | 喜欢简洁回答、不喜欢长篇大论 |
| identity | 用户身份 | 名字叫老李、在北京工作 |
| project | 项目相关 | 正在做 FluxPanel、用 Go 开发 |
| relationship | 关系习惯 | 叫我老李、像朋友一样说话 |
| instruction | 长期指令 | 每次回复带时间、用中文 |
| fact | 普通事实 | 有两个显示器、用 Mac |

## 重要度评分

| 分数 | 说明 |
|------|------|
| 9-10 | 核心信息：名字、项目名、关键偏好 |
| 7-8 | 重要信息：职业、技术栈、常用称呼 |
| 5-6 | 一般信息：普通偏好、一般事实 |
| 3-4 | 次要信息：可能变化的事实 |
| 1-2 | 临时信息：低价值参考 |

## 去重规则

创建记忆前必须先搜索：

1. 如果同 category 有相似内容的记忆 → 调用 memory_update
2. 如果不同 category 但内容相似 → 跳过，不重复创建
3. 只有完全新的信息才调用 memory_create

## 输出规则

- 不要直接说"记忆保存成功"或"已创建记忆"
- 返回工具结果后，让主人格自然回复用户
- 例如："行，我记住了" 而不是 "记忆创建成功"
