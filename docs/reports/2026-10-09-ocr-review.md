# OCR 审查报告（2026-10-09）

## 范围与执行

- 仓库：`/Users/zhoudai/Downloads/Projects/ai-platform`
- 输入范围：37 个变更文件；OCR 实际纳入 36 个。唯一未纳入项为 `deploy/production/production.env.example`，工具的扩展名过滤器不支持该文件；未绕过过滤。
- 工具：`ocr` v1.12.13，参数 `review --audience agent --effort high`；OCR 会话 `73587c60-fd62-498e-a820-bbc9df4765b7`。
- OCR 输出：36 个文件，23 条评论；约 13 分 9 秒完成，退出码 0。两次 HTTP 503 后请求恢复。
- 评审模型：根据本机 wrapper 的路由行为，推断本轮走 OpenCode Go 的 DeepSeek V4 Flash 主路由；输出未出现自动回退提示。此项为路由推断，不是服务端独立审计证据。
- 规则：本报告路径已加入排除项，避免后续 OCR 将报告自身作为输入。
- 处置范围：只对高、中优先级问题建立修复队列；低优先级意见按技能规则忽略，不列明细。

## 高风险

| ID | 位置 | 问题与影响 | 建议处理 | 状态 |
| --- | --- | --- | --- | --- |
| H-01 | `deploy/production/bridge-firewall.sh:6,19-25`；`deploy/production/docker-compose.yml:132` | 防火墙规则把 Bridge 原始目标地址写成宿主机 `<AI平台服务器内网地址>:3001`，而 HZC 通过 Docker 内网域名直接访问 Bridge 容器地址 `<Bridge容器内网地址>:3001`，且 Compose 不发布宿主机端口。按 Docker `DOCKER-USER` 的 DNAT 规则语义，该匹配不覆盖预期流量，隔离规则可能不生效。 | 规则已改为匹配 Bridge 容器地址 <Bridge容器内网地址>:3001；临时隔离测试确认放行与拒绝规则。生产服务器的真实网络路径仍需另行验证。 | ✅ 已修复：本地已修复并通过隔离脚本测试；未部署|

## 中风险

| ID | 位置 | 问题与影响 | 建议处理 | 状态 |
| --- | --- | --- | --- | --- |
| M-01 | `.project-workflow/scripts/verify_workflow_setup.py:62-70` | 检查器用自身文件位置查找必需脚本，而不是目标项目目录；检查其他项目时可能误报或漏报。 | 从目标项目的工作流目录检查必需脚本，并补一个跨目录回归用例。 | ✅ 已修复：本地已修复；回归测试通过；未提交|
| M-02 | `docs/部署与凭据说明.md:80` | 文档仍写 Bridge→New API 使用 HTTP，与当前生产 Compose 的 HTTPS 上游配置及后续验收记录冲突。 | 更新为当前已部署状态，并保留尚未完成的业务验收边界。 | ✅ 已修复：已依据当前生产核验记录更新；事实、验收边界检查通过|
| M-03 | `docs/部署与凭据说明.md:88` | 文档称 Bridge systemd 禁用、服务未启动，与当前生产台账中 enabled/active、readiness 通过的记录冲突。 | 更正文案为当前已验证的状态。 | ✅ 已修复：已按当前生产记录更正；服务状态与开关边界检查通过|
| M-04 | `main.go:283-290` | 内部 TLS HTTP Server 未设置请求头读取超时、空闲超时和最大请求头大小，可能被慢连接或过大请求头长期占用资源。 | 设置适当的标准库 Server 限制，并添加针对新 Server 配置的最小回归断言。 | ✅ 已修复：已设置请求头读取和空闲超时；请求头大小沿用 Go 标准库默认上限；地址覆盖及证书配置分支回归测试通过|
| M-05 | `deploy/production/生产对照核验.md:74-75`；`docs/生产服务器部署状态.json:145,149` | 同日注册开关记录同时出现 true 与 false，缺少清晰的时间/快照关联，读者无法判断当前有效值。 | 标注每次回读时间与先后，明确最新状态和未验证事项。 | ✅ 已修复：已明确较早与后续回读的顺序，并注明 TLS 发布后未复核；JSON 解析与顺序断言通过|
| M-06 | `.project-workflow/scripts/install_git_gate.sh:25-34` | 现有 hook 若没有执行权限，安装器只追加门禁但不补执行权限，提交时门禁仍可能不运行。 | 无论新建还是复用 hook，都确保其可执行；用临时仓库验证。 | ✅ 已修复：新增 hook 和已存在 hook 的路径均设置执行权限；临时 Git 仓库回归测试 2/2 通过|
| M-07 | `.project-workflow/scripts/install_git_gate.sh:20-22` | 提前退出检测只匹配单独一行的 `exit N`，可能漏掉条件/内联退出，随后把门禁追加到不可达代码之后。 | 无法安全解析 shell 控制流时，发现 exit 标记即拒绝自动追加；注释中的 exit 也可能触发保守拒绝 | ✅ 已修复：条件内联 exit 回归测试通过；限制已说明|
| M-08 | `.project-workflow/scripts/project_workflow_status.py:31-32` | `--initialize` 所需模板文件不存在；状态文件缺失时初始化会因读模板失败而中断。 | 提供模板，或从仓库现有模板来源初始化；加一个最小初始化测试。 | ✅ 已修复：已补齐初始化模板；临时目录初始化并保留已有状态的回归测试通过|
| M-09 | `deploy/production/bridge-firewall.sh:15-44` | 脚本只在手动执行时应用规则，未发现持久化或重启后重放配置；主机重启后规则状态无法保证。 | 采用项目现有 systemd 生命周期管理持久化，并验证重复应用/移除和重启恢复。 | ✅ 已修复：本地已新增防火墙 systemd 单元，并让 Bridge 服务依赖它；隔离脚本验证和单元关系自检通过。未安装到生产服务器；重启恢复需部署后验证|
| M-10 | `deploy/production/生产基线台账.md:32` | 台账仍称生产 Bridge→New API 上游为 HTTP，并据此禁止写商户 Key，与目前已部署 HTTPS 和凭证只读验证记录冲突。 | 更新当前状态和仍未通过的业务门禁。 | ✅ 已修复：已更新为 HTTPS、服务器受控配置和只读模型目录验证状态；注明推理、共享余额及扣费仍未验收；文档断言与差异检查通过|
| M-11 | `deploy/production/docker-compose.yml:74-76` | 日志宿主机目录挂到容器 `/app/logs`，但 New API 工作目录为 `/data`、默认日志路径为相对的 `./logs`；独立日志挂载可能没有承接实际日志。 | 将挂载目标对齐实际日志路径，并验证容器内写入映射到预期宿主机目录。 | ✅ 已修复：已将目标改为默认日志目录 `<AI平台日志目录>`；Docker Compose 解析及挂载映射断言通过；未部署变更|
| M-12 | `deploy/production/生产对照核验.md:80-84` | “收口规则”无日期且处于较新的历史段落之前，仍说 Bridge/商家接入未完成；与后续 2026-10-09 完成记录容易冲突。 | 给历史规则标注适用时间，或更新成当前待办状态。 | ✅ 已修复：已标注为 2026-10-08 首轮核验时的规则，并指向 2026-10-09 P08 后续记录；时间与内容断言、差异检查通过|
| M-13 | `deploy/production/生产部署执行清单.md:13-15` | P08 描述在 Bridge 后验收“共享余额和扣费”，但 P06 将真实模型调用后置；执行顺序可能让人误以为可在独立门禁前做会产生费用的验收。 | 明确 P08 当前只做无模型/无扣费校验，真实推理和扣费必须先通过 P06 并另行授权。 | ✅ 已修复：已将 P08 限定为 HZC AI 关闭状态下的健康、凭证和只读目录校验；真实推理、充值、共享余额及扣费需先通过 P06 并单独授权；门禁断言通过|
| M-14 | `AGENTS.md:201-204` | 工作流规则要求回写 `PROJECT.md`，仓库同时有根目录 `PROJECT.md` 与 `.project-workflow/PROJECT.md`，目标不明确。 | 明确唯一状态文件路径，避免阶段门禁写错文件。 | ✅ 已修复：已将测试资产和后续待办规则均指向 `.project-workflow/PROJECT.md`，并与工作流脚本路径一致；路径一致性断言通过|
| M-15 | `deploy/production/backup-postgres.sh:26-28,41-44` | `pg_dump` 失败或中断会留下 `.partial` 文件，而轮转只清理完整 `.dump`；重复失败可能积累占用备份盘空间。 | 用退出清理处理本次临时文件，并验证失败时不遗留部分文件。 | ✅ 已修复：已在确认无同名文件后安装 EXIT 清理，失败的模拟 `pg_dump` 留下部分输出时临时 dump 与校验文件均被清除；bash -n 和差异检查通过|


## 修复边界

本报告只记录代码审查发现。本轮依据负责人确认逐项处理高、中风险及可安全修复的低风险项；改动仅在本地仓库，未提交、推送或部署，未调用模型、执行扣费或改动生产服务器。Redis Secret 注入保持现状；Bridge 防火墙生产安装和 Docker 重启恢复仍列为待办。

## 二轮复审

- 工具：`ocr` v1.12.13；会话 `3bc2e665-1710-48b4-a63e-b58e5d9c4cfc`；39 个文件，17 条评论，约 15 分 7 秒。
- 复审有 2 个文件组请求超时（195 个请求中 2 个失败）；这两组仍需补审，不能将本轮视为完整通过。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R2-M-01 | 中 | `main.go:254-255` | 公共与内部 HTTPS 服务顺序关闭，共用同一期限会让前者耗尽内部服务排空时间。 | ✅ 已修复：已改为并发关闭；定向回归测试通过 |
| R2-M-02 | 中 | `.project-workflow/.task-checkpoint.json` | 工作流检查点是本地临时产物，可能误入提交并削弱结束门禁。 | ✅ 已修复：核实该文件未跟踪；已加入精确忽略规则，文件仍保留在本机；忽略状态与文件存在性检查通过 |
| R2-M-03 | 中 | `deploy/production/生产对照核验.md`、`生产基线台账.md`、`.project-workflow/PROJECT.md` | P08 防火墙规则部署与 Docker 重启恢复尚无生产证据，阶段条件可能被误读为已完成。 | ✅ 已修复：已明确本地候选未部署、无生产重启证据、访问限制待验收，并在待办、门禁与 Transition 标注未通过；状态断言通过 |

## 三轮复审

- 工具：`ocr` v1.12.13；会话 `869f5b39-1b0b-4528-8a49-1ee1e9879e91`；39 个文件全部完成，18 条评论；约 12 分 44 秒。OCR 记录 1 个 LLM 错误，但会话覆盖状态为 complete；不能将其描述为零请求错误。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R3-M-06 | 中 | `PROJECT.md:65` | “暂不建设生产发布”与已完成生产部署的事实冲突。 | ✅ 已修复：删除“生产发布”这一过时范围项；部署状态由工作流状态文件记录 |
| R3-M-07 | 中 | `PROJECT.md:67` | 根文档包含生产 IP、容器地址和网络名等公开拓扑信息。 | ✅ 已修复：删除具体拓扑；脱敏检查确认这些值不再出现在根文档 |
| R3-H-12 | 高 | `deploy/production/backup-postgres.sh:8-9` | 发布镜像标签不是纯十六进制，导致生产备份预检拒绝运行。 | ✅ 已修复：放宽为安全镜像标签字符集并继续拒绝路径穿越；使用真实格式标签完成隔离备份与轮转模拟 |
| R3-M-13 | 中 | `deploy/production/bridge-firewall.sh:7-8` | 防火墙目标地址与 Compose 漂移时规则可能静默失效。 | ✅ 已修复：应用前核对 Compose、容器和监听地址，失配时失败关闭；隔离脚本检查通过 |
| R3-M-14 | 中 | `deploy/production/bridge-firewall.sh:25` | Bridge 同时连接两个 Docker 网络，防火墙只限制一侧地址。 | ✅ 已修复：为两个网络地址设置一致的隔离规则；地址与规则断言通过 |
| R3-M-18 | 中 | `deploy/production/生产基线台账.md:9` | 生产台账包含公网地址、内部拓扑、绝对路径及证书信息。 | ✅ 已修复：对公开台账中的生产细节使用占位符；脱敏检查通过 |

## 四轮复审

- 工具：`ocr` v1.12.13；会话 `787f87e5-3a72-472b-90bc-23c825e10fe1`；39 个文件中 29 个完成、10 个因 LLM 请求超时未完成，记录 8 条评论。此轮本身为 partial，不能单独算作完整复审。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R4-M-02 | 中 | `docs/部署与凭据说明.md:0` | 文档把已部署的 Bridge 固定地址与尚未部署的防火墙规则合并描述为“候选未部署”。 | ✅ 已修复：分别说明固定地址与内网 HTTPS 已生效，待安装的是防火墙持久化单元；文档交叉断言通过 |
| R4-M-04 | 中 | `.dockerignore:17-18` | 宽泛 `*.local` 忽略规则会排除构建需要的 `Dockerfile.local`。 | ✅ 已修复：收窄为环境文件规则；Dockerfile 保留断言和 Compose 检查通过 |
| R4-M-05 | 中 | `.gitignore:43-44` | 环境文件忽略规则未覆盖通用 `*.env` 和 `.env.*`，且模板需保留可跟踪。 | ✅ 已修复：补齐忽略规则并显式放行示例模板；模式检查通过 |
| R4-M-08 | 中 | `.project-workflow/scripts/verify_workflow_gate.py:57-59` | 发布门禁把模板值 `未执行` 当作有效全量回归记录。 | ✅ 已修复：补审后改为拒绝所有以“未执行”开头的值，回归用例通过，详见 R4-S-05 |

## 四轮补审

- 工具：`ocr` v1.12.13；会话 `0caafb2a-9ea4-4575-b9ab-e8c57dfbba71`；24 个文件全部完成、8 条评论、约 18 分 4 秒。覆盖第四轮未完成文件及相关修复文件。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R4-S-01 | 中 | `.dockerignore:16` | `**/.env*` 不匹配 `production.env` 一类非点开头文件。 | ✅ 已修复：增加 `**/*.env`；保留 Dockerfile 的断言和 Compose 配置检查通过 |
| R4-S-02 | 中 | `bridge/main.go:245-246` | 以值拷贝复用 `tls.Certificate` 会复制内部同步状态并触发 `copylocks`。 | ✅ 已修复：使用 `TLSConfig.GetCertificate` 返回已加载证书指针；Bridge 测试和 `go vet ./bridge` 通过 |
| R4-S-05 | 中 | `.project-workflow/scripts/verify_workflow_gate.py:57` | “未执行；本轮只做定向验证”仍能绕过只检查裸值的发布门禁。 | ✅ 已修复：门禁拒绝所有以“未执行”开头的值；回归测试覆盖裸值、带解释占位值和真实验证结果 |
| R4-S-08 | 中 | `deploy/production/生产对照核验.md:121` | 2026-10-08 表仍显示 Bridge 未启用、凭证未配置，容易被误读为当前状态。 | ✅ 已修复：同一历史快照提示明确当前状态以 2026-10-09 记录为准 |

第四轮的 10 个超时文件在本次 24 文件补审中均得到 OCR 完成状态。补审后工作流脚本回归测试 12/12、自检、文档状态交叉断言及 `git diff --check` 通过；补审后新增的代码和文档改动仍需最终定向 OCR 复审。


## 五轮定向复审

- 工具：`ocr` v1.12.13；会话 `4cd1a20f-c8d6-4f28-b866-4d89044bcbc1`；9 个文件全部完成；OpenCode Go 预检 HTTP 429 后使用已批准的 DeepSeek 官方 API 路由。
- 本轮没有高、中优先级问题。状态文档中的占位符一致性更正已完成并通过脱敏断言；其余低优先级意见按技能规则忽略，不列明细。

## 六轮最终复审

- 工具：`ocr` v1.12.13；会话 `3ae13e51-45a2-48e8-825e-c743f2b05d2d`；2 个文件全部完成；OpenCode Go 预检 HTTP 429 后使用已批准的 DeepSeek 官方 API 路由。
- 本轮没有高、中优先级问题；低优先级意见按技能规则忽略，不列明细。

## 七轮增量复审（2026-10-10）

- 范围：只审当前主工作区；OCR 预览 49 个变更文件，实际完成 40/40 个审查项，9 个文件因既有过滤规则排除。两个嵌套脏 worktree 未纳入。
- 工具：`ocr` v1.12.13；参数 `review --audience agent --effort high`；会话 `1683ad3b-f1fc-4c60-99a9-ba964f9d1aba`；约 12 分 54 秒。OpenCode Go 预检 HTTP 429 后自动切换到已配置的 DeepSeek 官方 API 路由；会话日志记录 3 个 LLM 请求错误，40 个审查项仍全部完成。
- OCR 输出：13 条评论，其中 4 条中优先级、9 条低优先级；没有高优先级问题。低优先级意见按审查规则忽略。

| 编号 | 位置 | 问题与影响 | 处置计划 | 状态 |
| --- | --- | --- | --- | --- |
| R7-M-01 | `deploy/production/systemd/ai-platform-bridge.service:3-4` | Bridge 候选单元硬依赖尚未安装到生产服务器的防火墙单元；若先启用该候选单元，Bridge 会因依赖缺失而无法启动。 | 保留失败关闭依赖；P08 已明确先安装并验证防火墙单元，再安装或启用 Bridge 单元。当前候选单元未部署。 | ✅ 已澄清：依赖是避免防火墙未就绪时启动 Bridge 的安全门禁；顺序由部署清单和回归断言固定，生产安装仍待 P08 |
| R7-M-02 | `deploy/production/systemd/ai-platform-bridge.service:8-9`；`docs/部署与凭据说明.md:78` | `Type=oneshot` 与 `RemainAfterExit=yes` 使服务状态保持 `active (exited)`；手动停止容器后执行 `systemctl start` 不会重新创建容器，服务状态也不是健康检查。 | 将恢复说明改为 `systemctl restart`，并说明服务状态不能替代 Bridge readiness 检查。 | ✅ 已修复说明：文档已明确恢复命令及健康核验边界；定向回归断言通过 |
| R7-M-03 | `docs/生产服务器部署状态.json:175-176` | 权威状态账本遗漏防火墙安装及 Docker 重启恢复门禁，也未列全 DeepSeek 价源和本地价格编辑器尚未部署等 P08 阻塞项。 | 补齐当前未完成门禁、阻塞项和下一步动作，使其与部署清单及 `.project-workflow/PROJECT.md` 一致。 | ✅ 已修复：P08 阻塞项与下一步已补齐；JSON 解析和阻塞项断言通过 |
| R7-M-04 | `deploy/production/docker-compose.yml:79-82` | New API 的可写 `/data` bind mount 使位于其源目录内的 TLS 私钥也能通过 `/data/tls` 写入；另一个只读 mount 到 `/run/ai-platform/new-api-tls` 并不会封住该别名。 | 将证书源目录移出 New API 可写数据目录，并同步 Certbot 钩子、文档和回归断言。 | ✅ 已修复：Compose 与 Certbot 钩子统一使用 `/data/ai-platform/new-api-tls`；配置回归确认该源目录位于可写数据挂载之外，`hdzc`/`hzc` Compose 校验通过。生产证书迁移及服务重启未执行，已列入 P08 发布门禁 |

人工补审还发现 Bridge Compose 使用 `restart: "no"`；由于 systemd 以 detached `docker compose up -d` 启动且保持 `active (exited)`，容器异常退出后不会被 Docker 自动拉起。该项已本地改为 `restart: unless-stopped`，增加 Compose 配置回归并更新运维说明。

## 修复后最终复审（首次尝试失败记录）

- 工具：`ocr` v1.12.13；会话 `b0895c82-85eb-4506-b7ac-3e92980038ca`；预览选择 40 个文件，但 40/40 项均因 LLM 请求失败而未完成。OpenCode Go 预检返回 HTTP 429，自动切换的 DeepSeek 官方 API 对 57 个请求均返回 HTTP 402。
- 该次输出的 0 条评论不代表没有问题；自动最终复审未完成，不能据此判定 OCR 闭环通过。失败后未重试或更换供应商。
- 修复后的定向验证通过：工作流单测 17/17、`hdzc`/`hzc` Compose 配置、状态 JSON、自检、结束校验、Certbot 脚本语法和 `rtk git diff --check` 均通过。生产证书目录迁移、Bridge 防火墙安装和 Docker 重启恢复未执行，生产发布门禁仍为 `NEED_VERIFY`。本轮未提交、推送、部署、重启 HZC、调用模型或扣费。低优先级评论按规则未修复。

## R8 复审（恢复会话，2026-10-10）

- 范围：当前主工作区；预览 49 个变更文件，按既有规则审查 40 项；9 项按规则排除，两个嵌套 worktree 未纳入。
- 工具：`ocr` v1.12.13；参数 `review --audience agent --effort high`；会话 `af584402-ea9d-403e-b01f-339d5d9e20da`；耗时 15 分 43 秒。OpenCode Go 预检返回 HTTP 429，随后走已批准的 DeepSeek 官方 API 直连；单次请求取消被记录，OCR 最终报告 40/40 项完成、11 条评论。
- 分类：0 高、10 中。以下问题均有明确触发路径；当时中优先级修复待负责人确认，之后负责人已授权并完成修复，记录见本报告的 R8 问题修复记录。低优先级不列出。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R8-M-01 | 中 | `.project-workflow/references/project-workflow-state-template.md:9`；`.project-workflow/scripts/workflow_state.py:11`；`.project-workflow/scripts/verify_workflow_gate.py:35-36` | 模板把首个“结论”放在“项目”段，而解析器全局读取首次匹配；新增“当前门禁-结论”后可能读到旧值，造成发布误阻塞或门禁绕过。 | ✅ 已修复：模板与状态读取器统一按“当前门禁”段读取；发布门禁回归确认旧段 PASS 不能覆盖当前 NEED_VERIFY。 |
| R8-M-02 | 中 | `.project-workflow/scripts/install_git_gate.sh:35` | 现有 hook 若以 `exec` 替换进程，追加在后面的提交门禁永远不会执行，可能静默绕过门禁。 | ✅ 已修复：安装器遇到 `exit` 或 `exec` 控制流标记时拒绝自动追加；已有退出与新增 exec 回归测试通过。 |
| R8-M-03 | 中 | `bridge/main.go:42-47,267-269`；`docs/部署与凭据说明.md:78` | 两个 TLS 变量都缺失时 Bridge 会回退到明文 HTTP；生产配置漂移时可能明文转发商户 Key。 | ✅ 已修复候选配置：证书或私钥缺失、无效时 Bridge 拒绝启动，入口只使用 HTTPS；Bridge 定向测试、静态配置回归和 vet 通过。未部署生产。 |
| R8-M-04 | 中 | `deploy/production/bridge-firewall.sh:6-7,34-35` | 防火墙来源网段硬编码，未核对外部 Docker 网络的实际子网；网络变更后可能放行失效、Bridge 不可达。 | ✅ 已修复：从 Compose 取得外部网络名并读取 Docker 当前唯一 IPv4 子网；Bridge 固定地址不在该子网时拒绝应用。配置说明与定向测试通过。 |
| R8-M-05 | 中 | `deploy/production/backup-postgres.sh:32` | 临时文件只在 `EXIT` 清理；systemd 停止时的 `SIGTERM` 不保证执行该 trap，可能遗留 `.partial` 文件并持续占用备份盘。 | ✅ 已修复：为 HUP/INT/TERM 安装退出处理，由现有 EXIT 清理删除临时 dump 和校验文件；信号自检与脚本语法通过。 |
| R8-M-06 | 中 | `deploy/production/nginx/ai-platform.conf:31` | Nginx 未转发 WebSocket `Upgrade`/`Connection`；New API 的 `/v1/responses` 与 `/v1/realtime` 经门户域名无法完成握手。 | ✅ 已修复候选配置：Nginx 根据请求升级状态设置 `Connection` 并透传 `Upgrade`；配置断言通过，文档已同步。未在服务器执行 `nginx -t` 或握手验收。 |
| R8-M-07 | 中 | `.project-workflow/scripts/install_git_gate.sh:25-27` | 安装器只凭 marker 判定门禁已安装；marker 存在但门禁命令被删时仍报告成功，提交时会静默跳过校验。 | ✅ 已修复：marker 存在但精确门禁命令缺失时失败关闭；回归确认 hook 内容不被改写。 |
| R8-M-08 | 中 | `deploy/production/backup-postgres.sh:17-21` | 备份脚本依赖 `/data/ai-platform/releases/$release/docker-compose.yml`，而生产服务和清单使用固定 `/data/ai-platform/bridge-compose.yml`；路径不存在时每日备份会失败。 | ✅ 已修复候选配置：备份改用固定生产 Compose 路径，保留发布标签格式校验；配置回归和脚本语法通过。 |
| R8-M-09 | 中 | `deploy/production/systemd/ai-platform-bridge.service:3-4,10`；`deploy/production/systemd/ai-platform-bridge-firewall.service:3-4`；`deploy/production/docker-compose.yml:108` | Docker 重启时 `unless-stopped` 可能先自动拉起 Bridge，防火墙单元在 Docker active 后才应用，期间存在访问隔离规则未就绪的窗口。 | ✅ 已修复候选配置：Compose 禁用容器自启，由 systemd 前台监督 Compose；Bridge 单元依赖并排在防火墙单元之后，且随 Docker 重启。Compose profile 与监督配置回归通过；生产重启顺序仍待服务器验收。 |
| R8-M-10 | 中 | `deploy/production/systemd/ai-platform-backup.service:10-16` | 每日备份任务失败后没有重试或失败通知；单次失败会让最近成功备份超过 24 小时，偏离台账中的 RPO。 | ✅ 已修复候选配置：增加失败重试，每 5 分钟最多启动 4 次（含首次），6 小时后由每日 timer 再调度；说明补充失败检查命令与无通知边界。 |

- 本轮仅完成复审与报告回写，未改代码、运行代码测试或部署；P08 生产门禁仍为 `NEED_VERIFY`。

## R8 问题修复记录

### R8-M-01

- 状态：✅ 已完成
- 方法：工作流状态解析器支持按 Markdown 二级标题读取字段；状态展示、提交/发布门禁和结束校验均从“当前门禁”段读取“结论”，模板同步将结论和发布证据放入该段。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_verify_workflow_gate.py'` 与 `... -p 'test_project_workflow_status.py'`，2/2、2/2 通过。
- 证据：新增回归用例证明“项目”段里的旧 `PASS` 不能覆盖“当前门禁”段的 `NEED_VERIFY`；状态命令回读 `NEED_VERIFY`；模板初始化包含当前门禁段。
- 边界：只处理“当前门禁”下的结论读取，不重写其他工作流字段的通用查找规则。
- 结论：✅ 当前项目状态不会因前置同名字段误读而跳过或误阻发布；可进入下一个问题。

### R8-M-02

- 状态：✅ 已完成
- 方法：扩展安装器对现有 `pre-commit` 的保守控制流检测，将 `exec` 与 `exit` 一样交由人工合并。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_install_git_gate.py'`，5/5 通过；`sh -n .project-workflow/scripts/install_git_gate.sh` 通过。
- 证据：新增 `exec ./existing-check` 用例确认安装失败且不会追加门禁标记；原有条件内联 `exit` 用例仍通过。
- 边界：继续使用关键字检测，不尝试解析 shell 控制流；注释或字符串含 `exit`/`exec` 也可能保守拒绝。
- 结论：✅ 自动安装不会把提交门禁追加到 `exec` 替换之后；可进入下一个问题。

### R8-M-07

- 状态：✅ 已完成
- 方法：幂等分支同时校验 marker 和完整门禁命令；若 marker 存在但命令缺失，明确失败并保留原 hook。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_install_git_gate.py'`，6/6 通过；`sh -n .project-workflow/scripts/install_git_gate.sh` 通过。
- 证据：新增 marker-only hook 用例确认安装器非零退出、提示修复且 hook 字节内容保持不变；已安装完整门禁的用例仍通过。
- 边界：只接受安装器写入的完整命令行；手工改写格式需要人工核对，不自动猜测等价 shell 命令。
- 结论：✅ marker 不再单独作为门禁已安装的证据；可进入下一个问题。

### R8-M-04

- 状态：✅ 已完成
- 方法：防火墙 `apply` 路径读取 Compose 的外部网络名，再从 Docker 网络 IPAM 读取实际 IPv4 子网；要求唯一 IPv4 子网并验证固定 Bridge 地址落在其中，校验失败时不应用规则。`remove` 路径不依赖外部网络存在。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，7/7 通过；`bash -n deploy/production/bridge-firewall.sh`、标准库 IP 网段包含/排除自检和 `rtk git diff --check` 通过。
- 证据：回归检查确认子网来自 `docker network inspect`，不再写死 `172.*` 来源网段；运维说明记录单网段要求和失败关闭行为。
- 边界：本轮未连接生产 Docker 主机；实际网络名、IPAM 子网和部署后转发链路仍待 P08 服务器验收。
- 结论：✅ 网络变更会在规则应用前被验证，错误网段不会生成错误放行规则；可进入下一个问题。

### R8-M-09

- 状态：✅ 已完成
- 方法：Bridge 容器设置 `restart: "no"`，避免 Docker 在防火墙 systemd 单元完成前自动拉起容器；Bridge systemd 单元改为前台运行 `docker compose up --exit-code-from bridge`，并在进程退出时自动重启。保留对 firewall 单元的 `Requires`/`After`，并通过 `PartOf=docker.service` 传播 Docker 停止/重启。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，7/7 通过；Compose `hdzc` 和兼容 `hzc` 两个 profile 的 `config --quiet` 均通过；本机 Compose v5.1.2 帮助确认 `--exit-code-from` 和 `--no-deps` 选项存在。
- 证据：配置回归确认 Compose 不再自动重启 Bridge、systemd 以前台模式监督进程，且防火墙排序和 Docker 重启传播关系存在；部署说明已同步。
- 边界：本机没有 `systemd-analyze`，未在 Linux/systemd 或生产服务器上校验单元解析、Docker 重启期间的实际编排顺序和 readiness。
- 结论：✅ 候选配置消除了 Docker 自动先启 Bridge 的路径；真实重启恢复仍属 P08 部署验收项，可进入下一个问题。

### R8-M-03

- 状态：✅ 已完成
- 方法：Bridge 配置加载要求证书和私钥同时存在且可加载；入口只调用 `ListenAndServeTLS`，删除 HTTP 回退。部署说明同步记录 fail-closed 行为。
- 测试：`rtk test go test ./bridge` 通过；`rtk proxy go vet ./bridge` 通过；Bridge 部署配置回归 8/8、`rtk git diff --check` 通过。
- 证据：缺少两个 TLS 变量、仅有证书或仅有私钥时均有回归覆盖；有效证书加载和 TLS server 配置既有用例仍通过；Compose 配有只读证书路径。
- 边界：生产 Bridge 未重启；真实证书挂载、TLS readiness 和客户端证书链/主机名校验仍需服务器验收。
- 安全依据：OWASP ASVS v5.0.0 V12.3.3（内部 HTTP 服务间传输须加密且不得回退到明文）、Authentication Cheat Sheet、Web Service Security Cheat Sheet、Transport Layer Security Cheat Sheet；不代表完整 ASVS 验收。
- 结论：✅ 缺少 TLS 配置不会再让 Bridge 以明文方式启动或传递项目凭证；可进入下一个问题。

### R8-M-05

- 状态：✅ 已完成
- 方法：保留 EXIT 清理，并为 HUP、INT、TERM 安装退出处理，使 systemd 的 SIGTERM 停止路径最终执行同一清理。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，9/9 通过；`bash -n deploy/production/backup-postgres.sh` 通过。
- 证据：信号自检从脚本提取实际 trap 声明，在临时目录创建 `.partial` dump 与 checksum 后给测试 shell 发送 TERM，退出码为 1 且两个文件都被清除。
- 边界：SIGKILL、主机断电或文件系统故障无法运行 shell trap；本轮未在生产 systemd service 中发停止信号。
- 结论：✅ 可捕获的 systemd 停止信号不会遗留本次备份临时文件；可进入下一个问题。

### R8-M-08

- 状态：✅ 已完成
- 方法：备份脚本改为读取生产 systemd 和部署清单使用的固定 `/data/ai-platform/bridge-compose.yml`；保留 `AI_PLATFORM_RELEASE` 格式校验，Compose 仍从受控 `production.env` 读取镜像标签。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，10/10 通过；`bash -n deploy/production/backup-postgres.sh` 和 `rtk git diff --check` 通过。
- 证据：回归确认脚本不再拼接不存在的 `releases/$release/docker-compose.yml`，且仍拒绝无效发布标签；运维说明已记录输入文件路径。
- 边界：未在生产主机运行备份，也未读取生产 `production.env`；每日备份 service 的实际可用性仍需服务器验收。
- 结论：✅ 备份不再因按发布标签拼接 Compose 路径而找不到文件；可进入下一个问题。

### R8-M-06

- 状态：✅ 已完成
- 方法：在 AI 门户 `http` 上下文定义 `$connection_upgrade` 映射，在反代中透传 `Upgrade`，并按请求是否升级设置 `Connection`。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，11/11 通过；`rtk git diff --check` 通过。
- 证据：回归确认 map、两个反代请求头和 HTTPS/WSS 操作说明存在，并确认旧的空 `Connection` 设置已移除。
- 边界：本机未安装 Nginx，未执行 `nginx -t`；未在生产门户做 WebSocket 握手或 `/v1/responses`、`/v1/realtime` 端到端验收。
- 安全依据：OWASP ASVS v5.0.0 V4.4.1 要求 WebSocket 使用 WSS；不代表已经验证 New API 的 Origin 或会话策略。
- 结论：✅ Nginx 现在具备 WebSocket 升级请求转发配置，生产握手仍需 P08 验收；可进入下一个问题。

### R8-M-10

- 状态：✅ 已完成
- 方法：备份 `Type=oneshot` service 采用 `Restart=on-failure`、`RestartSec=5min`、`StartLimitBurst=4` 和 `StartLimitIntervalSec=6h`，形成最多 3 次自动重试；文档说明状态/journal 检查方式及当前无通知渠道。
- 测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，12/12 通过；`rtk git diff --check` 通过。读取 systemd 官方 service manpage，确认 oneshot 允许 `Restart=on-failure`，但不允许 `always`/`on-success`。
- 证据：配置回归断言了间隔、上限、失败重启策略和操作说明；未把定向配置断言描述为实际 systemd 重试运行。
- 边界：本机没有 `systemd-analyze`，也未触发真实备份故障；若 4 次尝试均失败，且随后每日 timer 再次失败，备份间隔仍可能超过 24 小时；当前没有失败通知。
- 结论：✅ 单次短暂故障具备有界重试；持续故障仍需管理员查看状态与日志，可进入下一个问题。


## R9 复审结果（2026-10-10）

- 范围：主工作区；40 个审查项全部完成。会话 b074c821-6fc7-4c7c-a425-763becda7bd9，参数 review --audience agent --effort high；OpenCode Go 预检 HTTP 429 后按既定配置切换至 DeepSeek 官方 API 直连。
- 结果：9 条评论，0 高、6 中、3 低；其中请求体读取超时在服务实现和测试各有一条重复评论，合并后为 5 个独立中优先级问题。低优先级按本轮范围保留。
- 复审范围和结论来自实际工具输出；生产环境未访问。

| 编号 | 优先级 | 文件 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R9-M-01 | 中 | bridge/main.go；bridge/main_test.go | Bridge 请求体读取未设置总时限；测试把零超时固定为预期值。 | ✅ 已修复并定向验证；细节见下方。 |
| R9-M-02 | 中 | docker-compose.yml；bridge/.env.hedezhongcheng.example；bridge/.env.edu-training.example；docs/部署与凭据说明.md | 本地 Bridge profile 未提供必需 TLS 配置，启动会失败，readiness 仍走 HTTP。 | ✅ 已修复并定向验证；细节见下方。 |
| R9-M-03 | 中 | web/src/features/system-settings/models/model-ratio-visual-editor.tsx；web/src/features/system-settings/models/model-pricing-sheet.tsx | 管理员旧入口仍查询已关闭的公开价格目录，可能继续触发 pricing is disabled。 | ✅ 已修复并定向验证；细节见下方。 |
| R9-M-04 | 中 | .project-workflow/PROJECT.md；docs/reports/2026-10-09-ocr-review.md | R8 发现轮和 R9 最终复审轮的名称与历史状态互相矛盾。 | ✅ 已修复并回归验证；细节见下方。 |
| R9-M-05 | 中 | .project-workflow/PROJECT.md | 将未执行的 systemd-analyze verify 记录为已通过。 | ✅ 已修正并回归验证；细节见下方。 |

### R9-M-01

- 状态：✅ 已完成
- 修复方法：Bridge HTTP server 设置 5 分钟 ReadTimeout 和 1 MiB MaxHeaderBytes；WriteTimeout 仍保持零值，以免截断流式上游响应。测试改为断言请求体读取上限、请求头上限和流式写入设置。
- 定向测试：rtk test go test ./bridge；rtk proxy go vet ./bridge。
- 验证证据：两条命令均退出码 0；Bridge 测试通过。
- 验证边界：未在生产或公网环境进行慢速请求探测；没有启动或重启任何服务。
- 结论：✅ 请求体读取有总时限且请求头上限明确；可进入下一个问题。

### R9-M-02

- 状态：✅ 已完成
- 修复方法：两个本地 Compose profile 都注入 Bridge TLS 证书/私钥容器路径，并只读挂载各自配置的宿主证书目录；健康检查改用 HTTPS。两个环境模板和部署说明补充证书文件位置、HTTPS readiness 与证书信任要求。
- 定向测试：rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'，13/13 通过；docker compose --env-file bridge/.env.hedezhongcheng.example --profile bridge config --quiet 和 edu-training 模板同命令均通过。
- 验证证据：测试确认两个 service 均声明 TLS 路径、只读证书目录与 HTTPS 健康检查；两个 Compose profile 配置退出码均为 0。
- 验证边界：没有生成或读取真实证书，没有启动容器；应用端对证书链的实际信任和 readiness 握手仍待本地/部署环境验证。Compose 输出旧版 version 字段弃用提示，不影响本次校验。
- 结论：✅ 本地 Bridge profiles 已提供强制 TLS 配置且不再使用 HTTP readiness；可进入下一个问题。

### R9-M-03

- 状态：✅ 已完成
- 修复方法：模型比率管理页停用公开价格目录查询；桌面编辑器显式关闭目录加载；移动端价格表单新增并透传该开关，避免子编辑器重新启用请求。
- 定向测试：bun run test src/features/system-settings/models/__tests__/model-ratio-table-selection.test.tsx，4/4 通过；bun run typecheck、目标文件 oxlint、oxfmt --check 均通过。
- 验证证据：桌面和移动端行为测试均确认打开编辑器时不会请求 /api/pricing。
- 验证边界：未登录生产后台验证；未修改公开价格目录功能或其开关；未部署。
- 结论：✅ 公开目录关闭时，管理员模型定价编辑流程不再触发该目录请求；可进入下一个问题。

### R9-M-04

- 状态：✅ 已完成
- 修复方法：把会话 af584402 标为 R8 复审，把发现新问题的会话 b074c821 标为 R9 复审；更新报告中“当时待确认、之后已获授权并修复”的时间顺序，以及工作流账本对应历史记录。
- 定向测试：rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_project_workflow_status.py'，3/3 通过。
- 验证证据：新增回归确认 R8 与 R9 使用不同标题，工作流账本不再把 R8 轮误标为待执行的最终复审。
- 验证边界：只校对并核对本轮新增或修改的审查轮次记录；未改动历史业务记录。
- 结论：✅ R8 发现轮与后续 R9 复审状态可区分；可进入下一个问题。

### R9-M-05

- 状态：✅ 已完成
- 修复方法：将工作流账本中“systemd-analyze verify 已通过”改为事实状态：本机未安装该工具，因此该命令未执行。
- 定向测试：rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_project_workflow_status.py'，4/4 通过。
- 验证证据：新增回归确认状态记录包含“未执行”，不再声称命令通过。
- 验证边界：未安装 systemd-analyze，未在 systemd 主机验证候选单元；线上既有服务运行证据属于以前的生产状态，不等同于本地候选单元校验。
- 结论：✅ 工作流账本不再声称未运行的 systemd 验证已通过；可进入最终复审。


## R10 最终复审（部分完成，2026-10-10）

- 范围：主工作区预览选中 43 项；会话 a5b4f569-599b-4c92-a147-a017ef32a66d，参数 review --audience agent --effort high，耗时 8 分 41 秒。OpenCode Go 预检 HTTP 429 后按既定配置切换到 DeepSeek 官方 API 直连。
- 结果：实际返回 6 条评论，其中 3 中、3 低；已返回评论中无高优先级项。另有 2 组 LLM 请求超时失败、1 组取消，不能将本轮视为完整通过；低优先级继续按范围保留。
- R10-M-01 至 R10-M-03 为需修复的中优先级问题。问题修复后需重新对主工作区完整复审。

| 编号 | 优先级 | 文件 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R10-M-01 | 中 | .project-workflow/PROJECT.md | R8/R9 已完成状态与历史表格中“R9 其余 4 项处理中”矛盾。 | ✅ 已修复并回归验证；细节见下方。 |
| R10-M-02 | 中 | bridge/.dockerignore | 本地 Bridge TLS 证书和私钥目录未从 Docker 构建上下文排除。 | ✅ 已修复并定向验证；细节见下方。 |
| R10-M-03 | 中 | bridge/Dockerfile.production；deploy/production/docker-compose.yml | BuildKit 默认目标平台与生产 Compose 固定 amd64 目标不够明确，跨架构构建可能产生不匹配镜像。 | ✅ 已修复并定向验证；细节见下方。 |

### R10-M-01

- 状态：✅ 已完成
- 修复方法：将 R8/R9 修复历史更新为 10 项与 5 项均已完成；另增独立 R10 部分复审记录，准确标明 3 项中优先级问题和 2 组超时。
- 定向测试：rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_project_workflow_status.py'，4/4 通过。
- 验证证据：测试核对 R8/R9 历史行的完成状态，并确认 R10 结果单独留档。
- 验证边界：R10 全量复审仍因两组模型请求超时和一组取消而不完整；修复剩余问题后须重新复审。
- 结论：✅ R8/R9 状态与 R10 当前发现已分别记录；可进入下一个问题。

### R10-M-02

- 状态：✅ 已完成
- 修复方法：在 Bridge 构建上下文忽略规则中排除本地 `certs/` 目录和所有 PEM 文件，避免证书、私钥进入镜像构建上下文。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，14/14 通过。
- 验证证据：新增回归断言要求忽略规则同时覆盖 `certs/` 和 `*.pem`。
- 验证边界：未读取或回显本地证书内容；本项只验证构建上下文忽略规则，未运行镜像构建。
- 结论：✅ Bridge 本地 TLS 证书与私钥不会进入 Docker 构建上下文；可进入下一个问题。

### R10-M-03

- 状态：✅ 已完成
- 修复方法：Go 构建阶段显式运行在 BuildKit 的构建平台；沿用 `TARGETOS`/`TARGETARCH` 交叉编译，并由生产 Compose 将 Bridge 运行平台固定为 `linux/amd64`。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，15/15 通过；`docker buildx build --builder orbstack --platform linux/amd64 --file bridge/Dockerfile.production --tag <临时标签> --load bridge` 构建成功，镜像检查为 `linux/amd64`。
- 验证证据：新增回归断言核对构建阶段、目标架构参数和 Compose 平台；本地临时镜像成功构建、核验后已移除。
- 验证边界：构建使用本机 OrbStack BuildKit，未在生产主机或生产流水线构建/发布；Compose 中的运行平台约束本轮未改变。
- 结论：✅ 本机跨平台构建能生成与生产 Compose 目标一致的 amd64 Bridge 镜像；可进入最终复审。

## R11 复审结果（部分完成，2026-10-10）

- 范围：主工作区最终预览 45 项；会话 `0148518d-8ade-4aca-865b-a35eef6bfda8`，高档、并发 2，耗时 14 分 52 秒。OpenCode Go 预检 HTTP 429 后按既定配置切换至 DeepSeek 官方 API 直连。
- 结果：返回 4 条中优先级评论；3/45 审查项失败，重试报告含 1 个已取消请求，因此本轮未完成，不能声称高优先级问题为零。OCR 子任务有一次 Certbot 文件读取失败；工作区核验确认两份钩子文件存在且可读。
- 本轮只记录中优先级问题；低优先级不进入修复范围。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R11-M-01 | 中 | .gitignore:46-48 | 缺少本地 Bridge TLS 证书目录忽略规则，可能将私钥加入 Git。 | ✅ 已修复：忽略 `bridge/certs/` 并验证 Git 不跟踪该目录文件。 |
| R11-M-02 | 中 | .dockerignore:14-17 | 根 Docker 构建上下文未排除 Bridge 本地 TLS 证书，根镜像构建可能携带私钥。 | ✅ 已修复：根 Docker 构建上下文排除 `/bridge/certs`。 |
| R11-M-03 | 中 | web/src/features/system-settings/models/model-ratio-visual-editor.tsx:152 | 停用公开价格目录查询后，任务模型名集合也变空，导致任务模型徽标、计数和筛选失效。 | ✅ 已修复：从管理员定价配置读取任务模型元数据。 |
| R11-M-04 | 中 | deploy/production/生产部署执行清单.md:12 | P05 标记已完成，但同一行又说明备份改动尚未部署，状态与验收描述矛盾。 | ✅ 已修复：将未部署的候选加固单独列出。 |

### R11-M-01 修复记录

- 状态：✅ 已完成
- 修复方法：在根 `.gitignore` 中加入 `bridge/certs/`，忽略本地 Bridge TLS 证书和私钥目录。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，16/16 通过。
- 验证证据：`git check-ignore -v --no-index bridge/certs/hedezhongcheng/privkey.pem` 命中根忽略规则；`git ls-files bridge/certs` 无已跟踪文件。
- 验证边界：忽略规则不会自动取消跟踪已纳入版本控制的文件；本次检查确认该目录没有已跟踪文件。
- 结论：✅ 本地 Bridge TLS 证书与私钥不会被 Git 新增跟踪；可进入下一个问题。

### R11-M-02 修复记录

- 状态：✅ 已完成
- 修复方法：在仓库根 `.dockerignore` 中加入 `/bridge/certs`，排除 New API 根构建上下文里的本地 Bridge 证书和私钥。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，17/17 通过；使用临时合成 PEM 文件验证 Docker 构建上下文无法 `COPY` 该路径。
- 验证证据：OrbStack BuildKit 对临时 `bridge/certs/.../privkey.pem` 的 `COPY` 按预期失败，且错误表明文件不在构建上下文中。
- 验证边界：仅用合成文件验证根上下文的忽略规则；未读取真实证书，也未构建或发布 New API 镜像。
- 结论：✅ 根 Docker 构建上下文会排除 Bridge 本地 TLS 证书与私钥；可进入下一个问题。

### R11-M-03 修复记录

- 状态：✅ 已完成
- 修复方法：视觉编辑器改用已有管理员 `useModelPricing()` 配置中的 `usage_schema` 识别任务模型；该查询与父级定价配置共享 React Query 缓存，不再依赖已关闭的公开 `/api/pricing` 目录。
- 定向测试：`bun run test src/features/system-settings/models/__tests__/model-ratio-table-selection.test.tsx`，5/5 通过；`bun run typecheck`、目标文件 oxlint 与 oxfmt 检查通过。
- 验证证据：新增管理员上下文回归用例覆盖已配置/未配置任务徽标、任务价格筛选结果，以及不请求公开价格目录。
- 验证边界：本地模拟管理员定价接口数据；未登录生产后台或修改实际价格配置。
- 结论：✅ 关闭公开目录时，任务模型识别、徽标和筛选继续使用管理员配置；可进入下一个问题。

### R11-M-04 修复记录

- 状态：✅ 已完成
- 修复方法：P05 完成条件只保留已实际通过的生产数据库、备份和隔离恢复证据；将 PID 文件名及 systemd 运行时限候选移至表格外，并明确尚未部署、不属于该阶段已完成的验收证据。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，18/18 通过。
- 验证证据：回归断言确认 P05 表格行不再包含“尚未部署”，且未部署候选有单独说明。
- 验证边界：只修正文档状态表达；没有检查或修改生产 systemd 服务。
- 结论：✅ P05 已完成证据与待部署候选加固已分开记录；可进入补审。

## R12 补充复审结果（主工作区，2026-10-10）

- 范围：针对 R11 修复后的改动和相关闭包审查 12 项；会话 `42238606-0cfe-4b74-a647-fed97003dc3d`，高档、并发 2，耗时 11 分钟。OpenCode Go 预检 HTTP 429 后，包装器按既定配置切换至 DeepSeek 官方 API 直连。
- 结果：12/12 项完成，返回 2 条中优先级意见，无高优先级意见，无失败项。低优先级不处理。

| 编号 | 优先级 | 文件:行号 | 问题 | 处置 |
| --- | --- | --- | --- | --- |
| R12-M-01 | 中 | .project-workflow/tests/test_bridge_deployment_config.py:35 | 防火墙测试只覆盖 Bridge 地址属于唯一子网的后一半条件，未锁定 IPv4 子网唯一性拒绝条件。 | ✅ 已修复：补充断言 `len(ipv4_subnets) != 1`，定向测试 18/18 通过。 |
| R12-M-02 | 中 | AGENTS.md:198 | `fast` 快速通道触发语法较宽，可能误入而绕过标准工作流。 | ✅ 已修复：仅接受紧跟冒号的 `fast:` 或 `fast：` 前缀，并保留原有任务资格限制。 |

### R12-M-01 修复记录

- 状态：✅ 已完成
- 修复方法：在 `test_firewall_uses_the_external_network_subnet` 中增加断言，确保测试同时锁定“IPv4 子网不唯一”与“Bridge 地址不属于子网”两种拒绝条件。
- 定向测试：`rtk test python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'`，18/18 通过。
- 验证证据：回归测试现在会在防火墙脚本去掉子网唯一性检查时失败。
- 验证边界：测试检查脚本条件表达式，没有在生产 Docker 网络或 iptables 上执行规则。
- 结论：✅ R12-M-01 已验证通过，可进入 R12-M-02。

### R12-M-02 修复记录

- 状态：✅ 已完成
- 修复方法：将快速通道入口收紧为 `fast:` 或 `fast：` 精确前缀，要求冒号紧跟 `fast`；保留原有小范围、无跟踪及排除高风险任务的条件。
- 定向验证：人工核对 `AGENTS.md` 入口语句，确认只接受两种冒号前缀且原有资格条件未删减；本项为对话规则文本，仓库脚本无法读取或验证用户消息前缀，因此未新增运行时检测代码。
- 验证证据：普通英文句子以 `fast` 开头但未带冒号时，不符合快速通道入口条件。
- 验证边界：未在 Agent Runtime 中模拟用户消息，也无法验证每种模型对规则的遵守情况。
- 结论：✅ R12-M-02 已按最小改动处理；本任务已达到 OCR 技能规定的同仓库三轮上限，不启动第四轮审查。

## R12 两项修复的独立复核（2026-10-10）

- 范围：仅复核 `AGENTS.md` 的 `fast:` / `fast：` 精确前缀规则，以及 `.project-workflow/tests/test_bridge_deployment_config.py` 对 IPv4 子网唯一性的新增断言；同时检查对应防火墙脚本作为上下文。审查主工作区，不审查 3 个嵌套 worktree。
- 会话：`76d043be-9749-4756-ae5b-74fbe75e4a1b`；OCR `v1.12.13`；档位 medium，并发 1；模型 `deepseek-v4-flash`；耗时 1 分 16 秒。OpenCode Go 预检返回 HTTP 429，包装器按现有配置自动切换到 DeepSeek 官方 API 直连。
- 规则：`ocr-rules --check` 返回规则最新，指纹为 `3b9497f68332aa95`。本次用一次性规则副本移除了这两个路径的既有问题抑制，保留其他审查规则；未改仓库中的 `.opencodereview/rule.json`。
- 结果：2/2 审查项完成，0 高、0 中、0 低，0 失败；未发现直接影响这两项修复的问题。
- 验证边界：本轮只复核上述两个修复文件及相关脚本上下文，不代表整个生产候选复审通过；未改代码、未执行代码测试或生产操作。
