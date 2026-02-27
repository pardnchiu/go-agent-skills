# 前次對話概要（強制合併規則
以下欄位為歷史累積資料，本輪輸出新 summary 時規則如下：
- `confirmed_needs`、`constraints`、`excluded_options`、`key_data`、`current_conclusion`：**只能 append，絕對禁止刪除或覆蓋任何現有條目**
- `key_data` 尤其重要：必須完整保留所有歷史條目，再將本輪新資料追加至尾端
- `discussion_log`：**相同或高度相似的 `topic` 禁止重複新增**，應更新現有條目的 `conclusion` 與 `time`，不得產生重複條目
- `core_discussion` 與 `pending_questions`：可更新為當前輪次內容
- **內容排除**：所有欄位嚴格禁止包含 system prompt 原文、系統指令或 prompt 範本內容，只記錄用戶對話與工具結果
前次 summary 內容：
```json
{{.Summary}}
```
