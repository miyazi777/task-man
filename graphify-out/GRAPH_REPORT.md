# Graph Report - /Users/tmiyajima/software/task-man  (2026-05-11)

## Corpus Check
- 59 files · ~63,253 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 952 nodes · 1598 edges · 75 communities (44 shown, 31 thin omitted)
- Extraction: 64% EXTRACTED · 36% INFERRED · 0% AMBIGUOUS · INFERRED: 582 edges (avg confidence: 0.81)
- Token cost: 353,802 input · 39,309 output

## Community Hubs (Navigation)
- [[_COMMUNITY_TUI App Shell & Overlays|TUI App Shell & Overlays]]
- [[_COMMUNITY_Navigation, Keys & Calendar|Navigation, Keys & Calendar]]
- [[_COMMUNITY_YAML Round-Trip Tests|YAML Round-Trip Tests]]
- [[_COMMUNITY_Overlay Rendering Helpers|Overlay Rendering Helpers]]
- [[_COMMUNITY_Task Move & Trash Ops|Task Move & Trash Ops]]
- [[_COMMUNITY_Field Definition Model|Field Definition Model]]
- [[_COMMUNITY_FieldDefList Operations|FieldDefList Operations]]
- [[_COMMUNITY_Attached File Preview|Attached File Preview]]
- [[_COMMUNITY_Hero Screenshot Concepts|Hero Screenshot Concepts]]
- [[_COMMUNITY_YAML Config Loaders|YAML Config Loaders]]
- [[_COMMUNITY_Tag Model & Tests|Tag Model & Tests]]
- [[_COMMUNITY_Layout Math & Ratios|Layout Math & Ratios]]
- [[_COMMUNITY_Popup Input Constructors|Popup Input Constructors]]
- [[_COMMUNITY_Tree Reordering Helpers|Tree Reordering Helpers]]
- [[_COMMUNITY_Detail Pane Rendering|Detail Pane Rendering]]
- [[_COMMUNITY_CLI Args Parsing|CLI Args Parsing]]
- [[_COMMUNITY_Storage File API|Storage File API]]
- [[_COMMUNITY_Issue Screenshot Annotations|Issue Screenshot Annotations]]
- [[_COMMUNITY_Status Model|Status Model]]
- [[_COMMUNITY_TUI Model Methods (EditorFiles)|TUI Model Methods (Editor/Files)]]
- [[_COMMUNITY_File Opener Resolution|File Opener Resolution]]
- [[_COMMUNITY_Repository & Reset Flow|Repository & Reset Flow]]
- [[_COMMUNITY_Default App Browser|Default App Browser]]
- [[_COMMUNITY_Extension Fields Design|Extension Fields Design]]
- [[_COMMUNITY_main.run Entry Flow|main.run Entry Flow]]
- [[_COMMUNITY_Task Validation|Task Validation]]
- [[_COMMUNITY_Row Building Tests|Row Building Tests]]
- [[_COMMUNITY_Detail Rows Build|Detail Rows Build]]
- [[_COMMUNITY_Footer & Hints Rendering|Footer & Hints Rendering]]
- [[_COMMUNITY_Editor Command Build|Editor Command Build]]
- [[_COMMUNITY_Field Value Mutations|Field Value Mutations]]
- [[_COMMUNITY_App Config Types|App Config Types]]
- [[_COMMUNITY_Args Parse Tests|Args Parse Tests]]
- [[_COMMUNITY_Editor Errors|Editor Errors]]
- [[_COMMUNITY_Next IDPosition|Next ID/Position]]
- [[_COMMUNITY_Layout HorizontalVertical Tests|Layout Horizontal/Vertical Tests]]
- [[_COMMUNITY_Layout Ratio Normalization|Layout Ratio Normalization]]
- [[_COMMUNITY_Row Navigation Helpers|Row Navigation Helpers]]
- [[_COMMUNITY_Layout Persistence|Layout Persistence]]
- [[_COMMUNITY_Subtask Nesting Rules|Subtask Nesting Rules]]
- [[_COMMUNITY_Config Struct Definitions|Config Struct Definitions]]
- [[_COMMUNITY_Layout Ratio Ensure|Layout Ratio Ensure]]
- [[_COMMUNITY_Aggregate Task Types|Aggregate Task Types]]
- [[_COMMUNITY_AssignMissingIDs Pattern|AssignMissingIDs Pattern]]
- [[_COMMUNITY_Repository Interface|Repository Interface]]
- [[_COMMUNITY_List ID Generation|List ID Generation]]
- [[_COMMUNITY_Field Type System|Field Type System]]
- [[_COMMUNITY_Field Date Helpers|Field Date Helpers]]
- [[_COMMUNITY_Field Type Cycle|Field Type Cycle]]
- [[_COMMUNITY_Field Type Bounds|Field Type Bounds]]
- [[_COMMUNITY_Task Children Check|Task Children Check]]
- [[_COMMUNITY_Detail Current Row|Detail Current Row]]
- [[_COMMUNITY_Cursor Clamp|Cursor Clamp]]
- [[_COMMUNITY_Cursor Follow Move|Cursor Follow Move]]
- [[_COMMUNITY_Current Status ID|Current Status ID]]
- [[_COMMUNITY_Tag Picker Open|Tag Picker Open]]
- [[_COMMUNITY_Field Edit Popup Open|Field Edit Popup Open]]
- [[_COMMUNITY_Toggle Task Tag|Toggle Task Tag]]
- [[_COMMUNITY_Detail Row Kind|Detail Row Kind]]
- [[_COMMUNITY_Catppuccin Palette|Catppuccin Palette]]
- [[_COMMUNITY_Calendar Month Shift|Calendar Month Shift]]
- [[_COMMUNITY_Application ID Gen|Application ID Gen]]
- [[_COMMUNITY_Opener Extension Check|Opener Extension Check]]
- [[_COMMUNITY_Forbidden Char Error|Forbidden Char Error]]
- [[_COMMUNITY_FieldDefList Type|FieldDefList Type]]
- [[_COMMUNITY_TaskField ByID|TaskField ByID]]
- [[_COMMUNITY_TaskField Remove|TaskField Remove]]
- [[_COMMUNITY_StatusList Type|StatusList Type]]
- [[_COMMUNITY_Default Statuses|Default Statuses]]
- [[_COMMUNITY_Status Color Setter|Status Color Setter]]
- [[_COMMUNITY_Tag Type|Tag Type]]
- [[_COMMUNITY_Tag Color Setter|Tag Color Setter]]
- [[_COMMUNITY_Tag Limit Rule|Tag Limit Rule]]
- [[_COMMUNITY_Module Path Decision|Module Path Decision]]
- [[_COMMUNITY_Popup Label Format|Popup Label Format]]

## God Nodes (most connected - your core abstractions)
1. `NewYAMLRepository()` - 37 edges
2. `DefaultStatuses()` - 29 edges
3. `Model` - 26 edges
4. `Model.View` - 22 edges
5. `truncate()` - 19 edges
6. `Model.handleKey` - 18 edges
7. `popupWidth()` - 17 edges
8. `PlaceOverlay` - 15 edges
9. `centerOverlay()` - 14 edges
10. `buildBorderRow()` - 14 edges

## Surprising Connections (you probably didn't know these)
- `data_base_directory` --references--> `storage.TaskDir`  [INFERRED]
  README.md → internal/storage/data.go
- `task-man implementation plan` --references--> `main.run`  [INFERRED]
  tasks/todo.md → cmd/task-man/main.go
- `--init flag (reset yaml)` --references--> `main.runInit`  [INFERRED]
  README.md → cmd/task-man/main.go
- `Multiple workspaces (-t)` --conceptually_related_to--> `cli.Parse`  [INFERRED]
  README.md → internal/cli/args.go
- `data_base_directory` --conceptually_related_to--> `storage.AppConfig`  [INFERRED]
  README.md → internal/storage/config.go

## Hyperedges (group relationships)
- **Main View rendering pipeline (list + detail + files + preview + footer)** — app_view, list_renderlist, detail_renderdetail, detail_renderfilenameslist, preview_renderpreview, footer_renderfooter [EXTRACTED 1.00]
- **Layout adjust mode (ensure, clamp, normalize, adjust h/v, apply to screen)** — layout_ensurelayoutratios, layout_clamplayoutratios, layout_normalizelayoutratios, layout_adjustlayouthorizontal, layout_adjustlayoutvertical, layout_applylayouttoscreen, layout_computelayoutdelta [EXTRACTED 1.00]
- **File open flow (resolve candidates, launch external app via $EDITOR or configured app)** — app_opencurrentfilewithdefault, app_opencurrentfilewithpicker, app_launchappforfile, editor_buildappcmd, app_editorfinishedmsg [EXTRACTED 1.00]
- **task mutation operations** — move_task_up, move_task_down, move_indent_task, move_outdent_task, trash_task, trash_restore_task, field_ops_set_field_value [INFERRED 0.85]
- **list CRUD lifecycle operations (add/rename/delete/move)** — status_list_insert_at, status_list_rename, status_list_delete, status_list_move_up, status_list_move_down, field_def_list_add_def, field_def_list_rename, field_def_list_delete, field_def_list_move_up, field_def_list_move_down, tag_list_add, tag_list_rename, tag_list_delete [INFERRED 0.85]
- **trash lifecycle ops** — trash_subtree_ids, trash_task, trash_restore_task, trash_root_id, trash_delete_subtree [INFERRED 0.95]

## Communities (75 total, 31 thin omitted)

### Community 0 - "TUI App Shell & Overlays"
Cohesion: 0.06
Nodes (70): buildBorderRow(), buildPaneDivider(), centerOverlay(), editReturnMode(), isSettingMode(), nextTagColor(), overlayConfirmPopup(), overlayErrorPopup() (+62 more)

### Community 1 - "Navigation, Keys & Calendar"
Cohesion: 0.06
Nodes (45): clampCursor(), NewModel(), openURLInBrowser(), formatFieldDate(), parseFieldDateOrToday(), renderSingleMonth(), shiftMonth(), wrapPopupContentRow() (+37 more)

### Community 2 - "YAML Round-Trip Tests"
Cohesion: 0.07
Nodes (53): NewYAMLRepository(), floatPtr(), TestYAMLApplicationsMissingFields(), TestYAMLApplicationsRoundTrip(), TestYAMLDataBaseDirectoryRoundTrip(), TestYAMLDuplicateTaskID(), TestYAMLEmptyFile(), TestYAMLFieldsAutoAssignID() (+45 more)

### Community 3 - "Overlay Rendering Helpers"
Cohesion: 0.05
Nodes (55): buildBorderRow, buildPaneDivider, centerOverlay, isSettingMode, overlayConfirmPopup, overlayErrorPopup, overlayFileOpenerPicker, overlayInputPopup (+47 more)

### Community 4 - "Task Move & Trash Ops"
Cohesion: 0.11
Nodes (44): IndentTask(), indexOf(), MoveTaskDown(), MoveTaskUp(), neighborStatusID(), OutdentTask(), peerIndexes(), ReassignTasksToFallback() (+36 more)

### Community 5 - "Field Definition Model"
Cohesion: 0.06
Nodes (20): loadFieldDefs(), IsKnownFieldType(), TestFieldDefListAddDef(), TestFieldDefListValidate(), TestValidateFieldDateValue(), TestValidateFieldURLValue(), TestValidateFieldURLValueLength(), ValidateFieldDateValue() (+12 more)

### Community 6 - "FieldDefList Operations"
Cohesion: 0.07
Nodes (46): FieldDefList.AddDef, FieldDefList.ByID, FieldDefList.DeleteByID, FieldDefList.MoveDown, FieldDefList.MoveUp, FieldDefList.RenameByID, FieldDefList.resequenceByOrder, FieldDefList.Sorted (+38 more)

### Community 7 - "Attached File Preview"
Cohesion: 0.1
Nodes (34): CreateFile(), CreateTaskData(), DeleteFile(), DeleteTaskData(), ListTaskFiles(), ReadTaskFile(), RemoveAllTaskData(), RenameFile() (+26 more)

### Community 8 - "Hero Screenshot Concepts"
Cohesion: 0.06
Nodes (34): task-man TUI Screenshot, Attached File: memo.md, Bubble Tea TUI Hero Image, Collapsible Status Groups, Color-coded Status Labels, Detail Files Section, Detail Field: ID, Detail Field: Status (+26 more)

### Community 9 - "YAML Config Loaders"
Cohesion: 0.08
Nodes (29): assignMissingPositions(), atomicWrite(), loadApplications(), loadFileOpeners(), loadLayout(), loadStatuses(), loadTags(), loadTasks() (+21 more)

### Community 10 - "Tag Model & Tests"
Cohesion: 0.09
Nodes (5): Tag, TestValidateTagNameChars(), ValidateTagNameChars(), TagList, TagNameForbiddenCharError

### Community 11 - "Layout Math & Ratios"
Cohesion: 0.16
Nodes (24): adjustLayoutHorizontal(), adjustLayoutVertical(), applyLayoutToScreen(), clampFloat(), clampLayoutRatios(), computeLayoutDelta(), ensureLayoutRatios(), isLayoutComplete() (+16 more)

### Community 12 - "Popup Input Constructors"
Cohesion: 0.09
Nodes (22): overlayCalendarPopup, renderSingleMonth, wrapPopupContentRow, newFieldURLValueInput, newFieldValueInput, newFileNameInput, newPopupInput, newTitleInput (+14 more)

### Community 13 - "Tree Reordering Helpers"
Cohesion: 0.24
Nodes (20): IndentTask, indexOf, neighborStatusID, OutdentTask, peerIndexes, ReassignTasksToFallback, renumberPositions, sortByPositionID (+12 more)

### Community 14 - "Detail Pane Rendering"
Cohesion: 0.17
Nodes (15): buildDetailRows(), detailLabelWidth(), padDetailLabel(), renderDetail(), renderDetailField(), renderFileNamesList(), renderTagsRow(), TestBuildDetailRowsNoFields() (+7 more)

### Community 15 - "CLI Args Parsing"
Cohesion: 0.18
Nodes (15): Args, EnsureFile(), expandHome(), Parse(), TestEnsureFileCreatesWhenAbsent(), TestEnsureFileExistingOK(), TestEnsureFileMustExistMissing(), TestParseDefault() (+7 more)

### Community 16 - "Storage File API"
Cohesion: 0.12
Nodes (18): storage.CreateFile, storage.CreateTaskData, storage.DeleteFile, storage.DeleteTaskData, storage.ListTaskFiles, storage.ReadTaskFile, storage.RenameFile, storage.TaskDir (+10 more)

### Community 17 - "Issue Screenshot Annotations"
Cohesion: 0.12
Nodes (17): Bubble Tea TUI two-pane layout (tree | Files), Files panel showing '(no files)', image1.png - task-man TUI screenshot, Design issue: PENDING status label is English while other statuses use Japanese, Keybinding footer (k/up, j/down, l/open, h/close, enter:detail, m:move, a:new/subtask, d:delete, o:operation, ;:prefix, q:quit), Status: 設計中 (Designing), Status: 実行済 (Executed), Status: 実装中 (Implementing) (+9 more)

### Community 19 - "TUI Model Methods (Editor/Files)"
Cohesion: 0.15
Nodes (16): Model.applyCollapseChange, Model.currentTask, editorFinishedMsg, Model.handleKey, Model.launchAppForFile, Model.openCurrentFileWithDefault, Model.openCurrentFileWithPicker, Model.persist (+8 more)

### Community 20 - "File Opener Resolution"
Cohesion: 0.21
Nodes (13): hasFileOpenerExtension(), nextApplicationID(), resolveDefaultApp(), resolveFileOpenerCandidates(), TestResolveDefaultAppCaseInsensitive(), TestResolveDefaultAppMatch(), TestResolveDefaultAppMissingDefault(), TestResolveDefaultAppNoOpener() (+5 more)

### Community 21 - "Repository & Reset Flow"
Cohesion: 0.17
Nodes (13): storage.RemoveAllTaskData, TestRemoveAllTaskData, LoadResult bundle (Repository sig change), main.runInit, --init flag (reset yaml), Single-file persistence, storage.Repository, storage.LoadResult (+5 more)

### Community 22 - "Default App Browser"
Cohesion: 0.17
Nodes (12): openURLInBrowser, resolveDefaultApp, resolveFileOpenerCandidates, TestResolveDefaultAppCaseInsensitive, TestResolveDefaultAppMatch, TestResolveDefaultAppMissingDefault, TestResolveDefaultAppNoOpener, TestResolveFileOpenerCandidatesCaseInsensitive (+4 more)

### Community 23 - "Extension Fields Design"
Cohesion: 0.24
Nodes (11): Extension fields plan rev.2, Top-level fields schema (rationale), Custom fields (text/date/URL), Custom statuses, YAMLRepository.Load, storage.loadFieldDefs, storage.loadStatuses, storage.loadTags (+3 more)

### Community 24 - "main.run Entry Flow"
Cohesion: 0.18
Nodes (11): cli.Args, TestEnsureFileCreatesWhenAbsent, TestEnsureFileExistingOK, TestEnsureFileMustExistMissing, cli.EnsureFile, bubbles/textinput width = m.Width+3, main.main, main.run (+3 more)

### Community 25 - "Task Validation"
Cohesion: 0.24
Nodes (6): ForbiddenCharError, Task, TestTaskValidate(), TestValidateTitleCharsForbidden(), TestValidateTitleCharsLength(), ValidateTitleChars()

### Community 26 - "Row Building Tests"
Cohesion: 0.2
Nodes (10): buildRows, TestBuildRowsCollapsed, TestBuildRowsMultiLevelNesting, TestBuildRowsOrderAndGrouping, TestBuildRowsPositionTieBreakerByID, TestBuildRowsSortedByPosition, TestBuildRowsSubtaskNesting, TestBuildRowsSubtaskSortedByPosition (+2 more)

### Community 27 - "Detail Rows Build"
Cohesion: 0.2
Nodes (10): NewModel, Model.withDetailRowsRebuilt, Model.withFilesRefreshed, buildDetailRows, TestBuildDetailRowsNoFields, TestBuildDetailRowsWithFields, keyMap, newKeyMap (+2 more)

### Community 28 - "Footer & Hints Rendering"
Cohesion: 0.25
Nodes (9): Model, detailRow, hintItem, renderFooter, renderHints, renderPopupHints, layoutFocus, Mode (+1 more)

### Community 29 - "Editor Command Build"
Cohesion: 0.39
Nodes (6): buildAppCmd(), TestBuildEditorCmdEnvVar(), TestBuildEditorCmdFallbackToEnv(), TestBuildEditorCmdLiteral(), TestBuildEditorCmdNoneConfigured(), TestBuildEditorCmdWithArgs()

### Community 30 - "Field Value Mutations"
Cohesion: 0.32
Nodes (6): PurgeRemovedFieldValues(), SetFieldValue(), TestPurgeRemovedFieldValues(), TestSetFieldValue(), TestValidateAll(), ValidateAll()

### Community 31 - "App Config Types"
Cohesion: 0.29
Nodes (8): storage.AppConfig, storage.Application, storage.FileOpener, data_base_directory, File opener, storage.loadApplications, storage.loadFileOpeners, TestYAMLApplicationsRoundTrip

### Community 32 - "Args Parse Tests"
Cohesion: 0.29
Nodes (8): TestParseDefault, TestParseExpandsTilde, TestParseInitFlag, TestParseInitWithTaskFlag, TestParseWithTaskFlag, cli.expandHome, cli.Parse, Multiple workspaces (-t)

### Community 33 - "Editor Errors"
Cohesion: 0.33
Nodes (7): buildAppCmd, ErrEditorNotConfigured, TestBuildEditorCmdEnvVar, TestBuildEditorCmdFallbackToEnv, TestBuildEditorCmdLiteral, TestBuildEditorCmdNoneConfigured, TestBuildEditorCmdWithArgs

### Community 35 - "Layout Horizontal/Vertical Tests"
Cohesion: 0.33
Nodes (6): adjustLayoutHorizontal, adjustLayoutVertical, TestAdjustLayoutHorizontal, TestAdjustLayoutVerticalDetailGrowsBeyondPreview, TestAdjustLayoutVerticalDetailGrowsFromPreview, TestAdjustLayoutVerticalNoOpForTaskList

### Community 36 - "Layout Ratio Normalization"
Cohesion: 0.4
Nodes (6): clampFloat, clampLayoutRatios, normalizeLayoutRatios, TestClampLayoutRatios, TestNormalizeLayoutRatiosPreservesProportions, TestNormalizeLayoutRatiosSumsTo1

### Community 37 - "Row Navigation Helpers"
Cohesion: 0.53
Nodes (6): Model.withRowsRebuilt, isNavigable, nextNavigable, prevNavigable, rowKind, TestNavigableSkipsSeparator

### Community 38 - "Layout Persistence"
Cohesion: 0.33
Nodes (6): storage.LayoutConfig, Top status hidden when width is small, Layout adjustment, storage.loadLayout, storage.marshalLayout, TestYAMLLayoutRoundTrip

### Community 39 - "Subtask Nesting Rules"
Cohesion: 0.33
Nodes (6): Subtasks (5-level nesting), Task ID = max+1 scheme, storage.assignMissingPositions, storage.loadTasks, TestYAMLSubtaskDepthExceeded, storage.validateParents

### Community 40 - "Config Struct Definitions"
Cohesion: 0.4
Nodes (4): AppConfig, Application, FileOpener, LayoutConfig

### Community 41 - "Layout Ratio Ensure"
Cohesion: 0.4
Nodes (5): ensureLayoutRatios, isLayoutComplete, TestEnsureLayoutRatiosFillsNil, TestEnsureLayoutRatiosPreservesExisting, TestIsLayoutComplete

### Community 42 - "Aggregate Task Types"
Cohesion: 0.5
Nodes (4): TaskFieldList, Status, TagList, Task

### Community 43 - "AssignMissingIDs Pattern"
Cohesion: 0.5
Nodes (4): FieldDefList.AssignMissingIDsAndPositions, TaskFieldList.AssignMissingIDs, StatusList.AssignMissingIDs, TagList.AssignMissingIDs

### Community 45 - "List ID Generation"
Cohesion: 1.0
Nodes (3): NextID, NextPosition, list_test

### Community 46 - "Field Type System"
Cohesion: 0.67
Nodes (3): FieldDef, TaskField, FieldType

## Knowledge Gaps
- **245 isolated node(s):** `keyMap`, `layoutFocus`, `editorFinishedMsg`, `hintItem`, `detailRowKind` (+240 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **31 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `run()` connect `CLI Args Parsing` to `Navigation, Keys & Calendar`, `YAML Round-Trip Tests`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **Why does `NewYAMLRepository()` connect `YAML Round-Trip Tests` to `YAML Config Loaders`, `CLI Args Parsing`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Why does `DefaultStatuses()` connect `YAML Round-Trip Tests` to `YAML Config Loaders`, `Status Model`, `Task Validation`, `CLI Args Parsing`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Are the 36 inferred relationships involving `NewYAMLRepository()` (e.g. with `run()` and `runInit()`) actually correct?**
  _`NewYAMLRepository()` has 36 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `DefaultStatuses()` (e.g. with `runInit()` and `TestYAMLRoundTrip()`) actually correct?**
  _`DefaultStatuses()` has 28 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Model.View` (e.g. with `twoPaneWidths` and `isLayoutComplete`) actually correct?**
  _`Model.View` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `truncate()` (e.g. with `.String()` and `previewLines()`) actually correct?**
  _`truncate()` has 17 INFERRED edges - model-reasoned connections that need verification._