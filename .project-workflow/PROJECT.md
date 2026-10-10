# 项目工作流状态

## 项目

- 名称：ai-platform
- 目标：在 HZC 当前生产服务器上独立部署 AI 平台，让平台管理员和 HZC 商家共用一个登录网址，并让 HZC 后台用户共享商家 AI 余额。
- 负责人：周到
- 当前阶段：7. 运行与优化 / 生产部署准备（生产门禁 P08）

## 当前任务状态（2026-10-10 生产部署配置审查与定向修复）

- 用户确认平台管理员与 HZC 商家共用 `https://<AI平台公网域名>`；平台按账号权限显示不同功能。
- HZC 后台用户通过内部 Bridge 调用 AI，共用一个商家企业计费账号和余额。Bridge 不作为公开登录入口。
- 两个项目在 `<AI平台公网地址>` 上独立运行；不复用 HZC 应用容器、配置、MySQL 或数据目录。本轮不使用 `edu-training`。
- P00、P01、P02 已完成。2026-10-08 两个权威 DNS 均回读到 `<AI平台公网域名> A <AI平台公网地址>`；证书续期模拟通过，并已启用 AI 专属 Nginx 配置。应用未发布时 AI 域名返回 503，占位状态与 HZC 管理端隔离；证据见 `deploy/production/生产对照核验.md`。
- P03 已完成：前端生产构建、固定 Go 1.26.1 的后端交叉编译、Bridge 定向测试、Compose 默认与当时的 HZC profile 配置校验及 gitleaks 秘密扫描通过；镜像归档上传并加载；摘要未留存，当前记录无法复核传输一致性；镜像当时尚未启动。根 Dockerfile 完整 BuildKit 构建在下载依赖时无输出并中止，改用同版本固定工具链离线编译并组装运行时镜像；详细证据和边界见 `deploy/production/生产对照核验.md`。
- P04 已完成：服务器 `<AI平台数据目录>/production.env` 由服务器生成 PostgreSQL、Redis 和会话随机秘密，文件权限为 `600 root:root`；只回读键名/是否设置，未读取或输出秘密值。Bridge 配置在 P08 完成商户映射配置，具体验收见下文。
- P05 已完成：独立 PostgreSQL 15.19、Redis 8.10.1 在 `/data` 健康运行；每日备份 timer 已启用、手动 service 退出码 0；New API 迁移后备份恢复到 tmpfs 隔离 PostgreSQL 成功，源与目标均为 40 张 public 表。备份与主库暂在同一服务器，真实充值前仍需异地备份和业务数据恢复验证。
- P07 已完成：New API、PostgreSQL、Redis 均 healthy；首个平台管理员已创建并只读核对到 1 个 root 管理员。AI 域名根路径现由独立 Nginx 直接跳转到 `/sign-in`，避免访客看到 New API 介绍页；登录页和 API 健康接口公网返回 200，HZC 原入口仍通过。支付、模型和 Bridge 暂不启用。
- P08 注册设置最近一次只读回读为 `false/false`；本轮发布 TLS 后未重新读取这两项，也未提交注册请求或检查注册 API 的拒绝结果。
- 商户接入准备：负责人确认商户账号所属用户分组为 `vip`；截图显示商户账号内 API Key 已启用且令牌分组为 `vip`，密钥值保持遮蔽，本轮未读取或写入服务器。
- P08 内网 HTTPS 已完成并验证：Bridge 使用独立的 Let’s Encrypt ECDSA 证书（`<BRIDGE_INTERNAL_FQDN>`，有效期至 `<证书有效期已核验>`），不共享 AI 门户的通配符私钥；Certbot DNS-01 临时 TXT 已清理，权威 DNS 没有该域名的 A/AAAA 记录。生产 Compose 将 `<BRIDGE_INTERNAL_FQDN>` 作为 `<HZC_DOCKER_NETWORK>` 网络别名指向 `<Bridge容器内网地址>`，证书目录只读挂载，Bridge 最低 TLS 版本为 1.2，宿主机不发布 `3001`。
- 从 HZC `deploy-api-1` 容器通过域名、证书链及主机名校验访问 `https://<BRIDGE_INTERNAL_FQDN>:3001/health/ready`；TLS 1.2 和 1.3 均返回 200。临时探针已清理。自动续期钩子已安装并验证初次复制，但未等待续期事件或对该新证书执行续期 dry-run。
- P08 Bridge→New API 内网 HTTPS 已部署：为 `<NEW_API_INTERNAL_FQDN>` 单独签发 ECDSA 证书，Certbot DNS-01 续期模拟通过；New API 使用 `ai-platform-new-api:tls-c69fcdd9cab1`，健康检查和内网 TLS 1.2/1.3 探针通过。证书续期钩子已安装；端口 `3001` 未发布到宿主机。
- TLS 上游发布步骤开始前已备份 AI 平台 PostgreSQL、原 Compose 与生产环境文件；当次比对确认只有 `AI_PLATFORM_RELEASE` 变化，未改 HZC Nginx、服务、配置或数据库。AI 门户与 HZC API 仍健康。
- 2026-10-09 在已打开的管理员浏览器会话中确认此前 `hedezhongcheng` 无项目凭证，随后签发并轮换到第 2 版；项目凭证写入 HZC 服务端 `<HZC配置目录>/production.env`，项目凭证校验返回 HTTP 204。凭证值未显示。服务器回读确认 `AiPlatform__Enabled=false`；未重启 HZC，运行中的 HZC 容器尚未加载新项目凭证。
- 商户 New API Key 已由负责人保存到 `<AI平台数据目录>/production.env`。脱敏回读确认项目 ID 正确、`hdzc-shared` 映射存在且唯一、文件权限为 `0600 root:root`；未读取或输出密钥。Bridge systemd 已启用并运行，容器在内网 HTTPS readiness 返回 HTTP 200；通过 Bridge 发起只读 `GET /v1/models` 返回 HTTP 200，验证项目凭证与商户映射可用。该请求只取模型目录，不调用模型；HZC AI 仍为 false，HZC 服务未重启。
- 本次配置前将 AI 平台和 HZC 的生产环境文件、Bridge Compose 与 systemd 单元备份到 `<AI平台备份目录>/hzc-bridge-20261009T031407Z`；备份文件权限为 0600。未启动 Bridge、未调用模型或执行扣费。
- 2026-10-09 按负责人确认，将 HZC A-02 管理端运营分析使用的固定企业标识从 `platform-ops` 改为 `hdzc-shared`；各招聘企业仍使用各自企业标识。同步更新 HZC 服务端代码、默认配置、测试和配置说明，以及本地 HZC 配置、AI 平台说明与 SSO 测试。本地 HZC 开发配置也已设为 `AiPlatform__Enabled=false`。生产 `<HZC配置目录>/production.env` 已在备份 `<AI平台备份目录>/hzc-enterprise-id-20261009T041551Z/hzc.production.env` 后写入新标识；脱敏回读为 `AiPlatform__EnterpriseId=hdzc-shared`、`AiPlatform__Enabled=false`，权限保持 `640`。未重启 HZC，`deploy-api-1` 仍在运行，进程尚未加载新配置。
- 本次命名子任务：生产 Compose 与 systemd 已统一使用标准 Bridge profile `hdzc`，旧 `hzc` profile 保留为兼容别名；内部 Compose 网络键为 `hdzc-internal`，底层 Docker 网络 `<HZC_DOCKER_NETWORK>`、网段、地址、端口和 Bridge 行为保持不变。systemd 已重载并仅重启 Bridge，备份位于 `<AI平台备份目录>/hdzc-profile-20261009T045804Z-870254`。
- 2026-10-09 当前接入请求：负责人希望只配置 DeepSeek V4 Flash 并按官方现行价格计费。管理员页面已读到 `deepseek-v4-flash` 的已保存表达式为零价。上游价格同步选择内置 DeepSeek 源后，请求 `/api/pricing` 返回 `401 Authorization Required`，没有可应用的价格；同步请求未复用渠道密钥，未读取、复制或发送密钥，也未修改价目。官方文档列出 `/models` 模型列表接口，未找到机器可读价格接口。不得将已保存密钥发送到未文档化的价格路径；自动导入需先确认可用的官方价格源及安全认证方式。计价表达式仍不含年份和中国法定节假日日历。
- 后台价格编辑器错误已在本地修复：管理员面板不再依赖公开价格目录。代码和定向验证完成，但生产发布门禁仍为 `NEED_VERIFY`，所以线上页面尚未获得此修复。
- 预上线备份暂放服务器数据盘上的独立目录；服务器或数据盘故障可能同时影响主数据和备份。真实充值开放前必须增加异地备份并完成恢复验证。真实支付和真实模型调用保持关闭。
- 本轮通过受控 SSH 配置 AI 独立 Nginx 站点、镜像、秘密文件、PostgreSQL/Redis、备份定时任务并发布 New API；随后仅调整 AI 站点的 `/` 路径，使其跳转到 `/sign-in`。该阶段 HZC 站点配置、服务与数据库未改动，`deploy-api-1` 保持 Up。

## 测试资产

- OCR 三轮问题处理顺序：P0 备份标签校验拒绝实际发布标签（已修并验证）；P1 Bridge 双网卡暴露/防火墙失配、生产明细进入公开仓库、备份并发互相覆盖、工作流快照解析与清理（均已修并定向验证）；P2 发布门禁步骤、TLS 可选行为、商户映射示例、历史表格格式和 Python 缓存忽略（均已修并核验）。定价示例提示缺失属于关闭公开模型广场后的非计费功能变化；Go 标准库默认已有 1 MiB 请求头限制，不重复设置。
- 本次变更：New API 增加可选的内网 HTTPS 监听（TLS 最低 1.2）；生产 Compose 使用 `<NEW_API_INTERNAL_FQDN>:3001` 作为 Bridge 上游、增加 Docker 内网别名和只读证书挂载；新增 Certbot 专用证书部署钩子。HZC A-02 固定企业标识已更新为 `hdzc-shared`。Bridge 商户映射、内网 HTTPS 与命名已部署。HZC AI 最近一次有据的服务器回读为 2026-10-09 `false`；本轮未重新连接服务器复核。DeepSeek 通道和价目尚未配置。 本轮 OCR 候选改动：Bridge 只绑定 HZC 网络地址，防火墙在 Compose 地址、监听地址或自身目标失配时拒绝应用；备份脚本允许实际发布标签、输出可诊断预检错误并以 PID 区分同秒文件，systemd 候选设置 1 小时上限；状态账本已替换具体生产地址、目录、域名、网络及摘要为占位符。未改生产服务器。
- R8 的 10 项与 R9 的 5 个独立中优先级问题均已修复并逐项验证；R10 的 3 个、R11 的 4 个中优先级问题及 R12 的 2 个中优先级意见均已处理并完成对应验证。R12 覆盖此前 R11 未完成的 3 项；R12 修复后未再经 OCR 复审，本轮达到三轮上限，不能声称最终 OCR 审查通过。
- 最小验证：`CGO_ENABLED=0 rtk test go test . -run TestNewInternalTLSServer -count=1`、`rtk test go test ./bridge`、Certbot 部署钩子 `sh -n`、`rtk git diff --check` 均通过；Go 1.26.1 离线构建 linux/amd64 镜像成功。Certbot DNS-01 续期模拟通过；内网临时探针在 TLS 1.2、1.3 下均通过证书链、主机名和 `/api/status` 检查。生产 New API healthy，宿主机仅映射 `127.0.0.1:3000`；AI 门户/API 为 200，HZC 管理入口为 302、HZC API 为 200。2026-10-09 项目凭证校验返回 HTTP 204；脱敏回读确认 HZC AI 为 false、企业标识为 `hdzc-shared`，未重启 HZC。商户映射仅确认存在且唯一、未读取密钥；文件权限 `0600 root:root`。生产 Compose quiet validation 通过；Bridge systemd enabled/active、容器运行，内网 HTTPS readiness HTTP 200；只读 `GET /v1/models` 返回 HTTP 200 且响应有效。未调用模型、扣费或重启 HZC。HZC A-02 定向 .NET 测试 `8/8`、API build `0` warning / `0` error；AI 平台普通本机 Go 测试因缺少系统 C 头文件失败，`CGO_ENABLED=0` 定向测试通过。安全依据：OWASP ASVS 5.0.0 V12.1.1、Web Service Security Cheat Sheet、Transport Layer Security Cheat Sheet；不代表完整 ASVS 验收。证据见 `deploy/production/生产对照核验.md`。 第三轮定向验证：工作流单测 9/9、前端 typecheck 与目标文件 oxlint 通过；Compose JSON 与 Bridge 监听/防火墙地址一致；备份脚本语法、标签/路径拒绝、缺失输入诊断通过；状态 JSON 和文档脱敏断言通过。Go 官方文档确认 MaxHeaderBytes 为 0 时使用 1 MiB 默认值。systemd-analyze 未安装。第四轮 OCR 选中 39 个文件，29 个完成、10 个因 LLM 请求超时失败，共 8 条评论，已全部处理：Bridge TLS 复用预加载证书，`rtk test go test ./bridge` 通过；发布门禁拒绝“未执行”占位，全量工作流回归 12/12 通过。此轮 OCR 仍有 10 个文件因 LLM 请求超时未完成，需补审后才能判定完整通过；四个工作流入口共用向上查找状态文件的解析器，嵌套目录下状态、提交门禁和开始快照验证通过；工作流脚本回归测试 12/12 通过。提交钩子安装器遵循 `core.hooksPath`，钩子运行时解析仓库根目录，shell 语法检查和回归测试 4/4 通过；部署说明已更正固定地址绑定已生效、防火墙持久化单元未部署，交叉状态断言通过；生产台账已移除摘要未留存时的哈希一致性主张，JSON 解析与摘要声明断言通过；Docker 忽略规则覆盖 `.env*` 与 `*.env`，保留 Dockerfile.local；Git 忽略环境变量文件并放行模板，模式断言和 `docker compose config --quiet` 通过。补审首批 4/24 个文件新增 3 条意见：Docker 忽略规则已补 `*.env`；Bridge 改为通过 `GetCertificate` 复用同一证书指针，`go test ./bridge` 与 `go vet ./bridge` 通过。请求头上限意见未改代码：Go 官方 `net/http` 文档明确默认限制为 1 MiB，显式写入同值不改变行为。
  - R8 工作流测试 26/26、Bridge Go 测试与 vet、Shell 语法、Compose 双 profile 配置及 diff 检查通过。R9-M-01 的 Bridge Go 测试与 vet 通过；生产 Nginx/systemd/容器运行未验证。
  - R9-M-01 定向验证：rtk test go test ./bridge 与 rtk proxy go vet ./bridge 均通过；未做生产慢速请求探测。
  - R9-M-02 定向验证：Bridge 部署配置测试 13/13 通过；HZC 与 edu-training Compose config --quiet 均通过。未生成证书或启动容器。
  - R9-M-03 定向验证：桌面/移动端定向测试 4/4、typecheck、目标文件 oxlint 和格式检查通过；未做线上验收。
  - R9-M-04 定向验证：工作流状态回归 3/3 通过；报告与账本轮次名称一致。
  - R9-M-05 定向验证：工作流状态回归 4/4 通过；未执行 systemd-analyze verify。
  - R10 复审前回归：工作流测试 29/29、Bridge Go 测试/vet、前端目标测试 4/4/typecheck/目标文件 lint/格式、两个本地及两个生产 Compose profile 校验、Shell 语法与 diff 检查均通过；OCR 规则指纹为最新。未执行生产部署或实际证书握手。
  - R10 部分复审：43 个选中项返回 6 条评论（3 中、3 低），另 2 组超时和 1 组取消；不视为完整通过。
  - R10-M-02 定向验证：Bridge 构建上下文忽略规则测试 14/14 通过，覆盖 `certs/` 与 `*.pem`；未运行镜像构建。
  - R10-M-03 定向验证：Bridge 部署配置测试 15/15 通过；OrbStack BuildKit 成功构建 linux/amd64 镜像并检查架构，临时镜像已移除。
  - R11 部分复审：45 个选中项中 42 项完成，4 条中优先级评论，3 项失败；不能视为完整审查。问题清单见审查报告，低优先级不处理。
  - R11-M-01 定向验证：忽略规则回归 16/16 通过；`git check-ignore` 命中 `bridge/certs/`，Git 当前未跟踪该目录文件。
  - R11-M-02 定向验证：忽略规则回归 17/17 通过；OrbStack BuildKit 使用临时合成 PEM 验证根 `.dockerignore` 会排除证书路径，未读取真实证书或构建 New API 镜像。
  - R11-M-03 定向验证：管理员模型比率视觉编辑器测试 5/5、typecheck、目标文件 lint/format 通过；任务元数据来自管理员配置，未请求公开 `/api/pricing`。
  - R11-M-04 定向验证：生产清单回归 18/18 通过；P05 完成条件与未部署的本地候选加固分开记录，未检查生产 systemd。
  - R12 补充复审：12/12 项完成，新增 2 个中优先级意见，无高优先级意见；覆盖此前 R11 未完成的 3 项。
  - R12-M-01 定向验证：`test_bridge_deployment_config.py` 18/18 通过；断言同时覆盖唯一 IPv4 子网和 Bridge 地址归属条件，未在生产网络执行防火墙规则。
  - R12-M-02 文本核对：快速通道仅接受 `fast:` / `fast：` 精确前缀，原有资格限制保留；仓库脚本无法读取对话消息，因此未新增运行时检测代码。
  - R12 结束状态：R11 未完成的 3 项已由 R12 覆盖，R12 两项中优先级意见均已处理；遵守 OCR 技能三轮上限，未对修复再启动第四轮 OCR。工作流单测 34/34、Bridge 配置定向测试 18/18、前端目标测试 5/5、前端 typecheck、规则校验与差异检查通过。
- 本轮增量审查与验证：OCR 会话 `1683ad3b-f1fc-4c60-99a9-ba964f9d1aba` 完成主工作区 40/40 审查项，13 条评论（4 中、9 低；低优先级未修）；3 个 LLM 请求错误已记录，不影响审查项完成状态。处置防火墙依赖顺序、Bridge systemd 恢复说明、P08 状态账本和 New API TLS 证书挂载问题。修复后 OCR 复审会话 `b0895c82-85eb-4506-b7ac-3e92980038ca` 的 40 个审查项全部因请求 HTTP 402 失败，不能当作零问题结果；恢复服务后仍须复审。工作流单测 17/17、`hdzc`/`hzc` Compose 配置、状态 JSON 解析、自检、结束校验、Certbot 脚本语法和 `rtk git diff --check` 通过。生产证书目录迁移、Bridge 防火墙安装和 Docker 重启恢复仍未执行。未提交、推送、部署、重启 HZC、调用模型或扣费。
- R8 复审：会话 `af584402-ea9d-403e-b01f-339d5d9e20da`；预览 49 个变更文件，审查 40/40 项，0 高、10 条中优先级问题；OpenCode Go 预检 HTTP 429 后使用已批准的 DeepSeek 官方 API 直连，记录 1 次请求取消。该轮产生的 R8 问题已全部修复并验证；本轮后续复审与各轮处置见 `docs/reports/2026-10-09-ocr-review.md`。
  - R8-M-01：解析器按“当前门禁”段读取结论，模板同步调整；门禁与状态脚本定向测试均 2/2 通过，旧段 `PASS` 无法覆盖当前 `NEED_VERIFY`。
  - R8-M-02：安装器遇到 `exec` 与 `exit` 标记时拒绝追加门禁；安装器回归 5/5、`sh -n` 通过。
  - R8-M-07：幂等分支要求 marker 和完整门禁命令同时存在；marker-only 情况失败关闭，安装器回归 6/6、`sh -n` 通过。
  - R8-M-04：防火墙应用前读取外部 Docker 网络实际 IPv4 子网并校验固定 Bridge 地址；配置回归 7/7、脚本语法和网段包含/排除自检通过，未连接生产 Docker。
  - R8-M-09：Bridge 改由排在防火墙之后的 systemd 前台监督，Compose 不再自动提前启动；配置回归 7/7、`hdzc`/`hzc` Compose 解析通过。未运行 systemd-analyze 或生产 Docker 重启。
  - R8-M-03：Bridge 缺少 TLS 证书或私钥时拒绝启动，删除 HTTP 回退；`go test ./bridge`、`go vet ./bridge`、部署配置回归 8/8 通过。按 OWASP ASVS v5.0.0 V12.3.3 和相关 Cheat Sheet 核验；生产 TLS readiness 未执行。
  - R8-M-05：备份脚本为 HUP/INT/TERM 增加退出处理；信号自检确认 `.partial` dump 与 checksum 删除，部署配置回归 9/9、`bash -n` 通过；未在生产 systemd 停止服务。
  - R8-M-08：备份脚本切换到 systemd 使用的固定 Compose 路径并保留发布标签校验；部署配置回归 10/10、`bash -n` 和差异检查通过；未执行生产备份。
  - R8-M-06：AI 门户 Nginx 增加 WebSocket Upgrade/Connection 转发并说明 HTTPS/WSS；部署配置回归 11/11 通过。Nginx 未安装，本地语法和生产 WebSocket 握手未验证。
  - R8-M-10：备份 service 配置 5 分钟重试，6 小时内最多 4 次含首次；部署配置回归 12/12 通过。systemd-analyze 和生产故障重试未执行；持续失败可能继续突破 24 小时 RPO，且未配置失败通知。
- 本次配置命名验证：服务器候选 Compose 的 `hdzc` 与兼容 `hzc` profile 均通过 `docker compose config --quiet`；本机未安装 `systemd-analyze`，未执行 `systemd-analyze verify`。部署后两个 profile 均解析到同一 Bridge 容器，systemd 使用 `hdzc`，Bridge readiness HTTP 200。HZC AI 仍为 false，HZC 容器 ID 与启动时间未变；工作流结束校验通过。
- 全量回归：未执行；本轮只做生产配置、Compose、Bridge readiness 与只读模型目录检查，没有运行 New API 全量回归或 HZC 业务验收；平台已有 1 个 root 管理员，本轮没有创建商家账号、充值或余额记录。
- OCR 补审收口：第三轮会话 `869f5b39-1b0b-4528-8a49-1ee1e9879e91` 选中 39 个文件、18 条评论、全部文件完成（1 个 LLM 错误）；第四轮 `787f87e5-3a72-472b-90bc-23c825e10fe1` 选中 39 个文件、29 个完成、10 个请求超时、8 条评论；补审 `0caafb2a-9ea4-4575-b9ab-e8c57dfbba71` 选中 24 个文件、全部完成、8 条评论。补审发现并修复发布门禁可被“未执行；说明文字”绕过的问题，`read_field` 和状态路径改由共享模块维护；部署文档区分 Bridge 服务已安装与防火墙持久化单元未安装，2026-10-08 状态表标注为已被 2026-10-09 记录取代。补审中重复建议显式设置 1 MiB `MaxHeaderBytes` 未采纳：Go `net/http` 在零值时已有相同默认上限。工作流脚本回归测试 12/12、自检、文档状态交叉断言及 `git diff --check` 均通过；第四轮 10 个超时文件已在 24 文件补审中完成，第五轮 9 文件和第六轮 2 文件定向复审均完成，无高/中优先级问题；状态文档占位符一致性断言通过。低优先级按 ocr-review-loop 规则不进修复队列；报告仅保留高/中问题，已完成项用 ✅ 标记。
- DeepSeek 配置验证：管理员浏览器已连接；内置 DeepSeek 上游价格源返回 HTTP 401，价格列表 0 项且“应用同步”禁用。读取到 `deepseek-v4-flash` 已保存表达式 `tier("base", p * 0 + c * 0)`；未修改价目、未调用模型、未产生扣费，也未读取或发送渠道密钥。自动导入仍需解决官方价格源和节假日分时规则。
- OCR H-01 本地修复：Bridge 防火墙规则改为匹配容器地址 <Bridge容器内网地址>:3001。隔离假 iptables 调用验证了 HZC 网段放行与其他来源拒绝规则；bash -n 与 rtk git diff --check 通过。未在服务器应用规则；真实转发链路仍需服务器侧验证。
- OCR M-01 本地修复：工作流自检器现从目标项目的 .project-workflow/scripts 检查必需脚本；新增临时目标仓库回归测试。unittest 1/1 与本仓库自检通过，rtk git diff --check 通过；待负责人确认后提交。
- OCR M-02：部署说明已更新 Bridge→New API 的 HTTPS 当前状态、证书配置和业务验收边界。基于生产核验记录检查文案与状态一致，M-02 文档断言及 rtk git diff --check 通过。
- OCR M-03：部署说明已更正 Bridge systemd 当前为 enabled/active、readiness 通过，并明确 HZC AI 仍关闭及业务验收边界。文档断言和 rtk git diff --check 通过。
- OCR M-04：内部 TLS HTTP Server 设置 10 秒请求头超时和 120 秒空闲超时；请求头大小沿用 Go 标准库默认上限，不再写入无效的默认值覆盖。测试覆盖默认/自定义地址、两种不完整证书配置及证书加载错误保留；定向 Go 测试、gofmt 与 `rtk git diff --check` 通过。
- OCR M-05：注册状态记录已区分较早 true 与同日后续 false 回读，并注明 TLS 发布后未复核及未验证注册拒绝行为。JSON 解析、证据顺序断言和 rtk git diff --check 通过。
- OCR M-06：Git hook 安装器现在也会为已有门禁和追加后 hook 设置执行权限。临时 Git 仓库回归测试 2/2、bash -n 与 rtk git diff --check 通过。
- OCR M-07：Git hook 安装器现遇到 exit 标记时拒绝自动追加，避免条件/内联提前退出绕过门禁；回归测试覆盖条件内联退出。bash -n、测试 3/3 与 rtk git diff --check 通过。该保守检测也可能匹配注释中的 exit。
- OCR M-08：已新增工作流状态初始化模板；定向测试验证模板创建、项目名替换和已有状态不被覆盖。测试 1/1、脚本 --help 与 rtk git diff --check 通过。
- OCR M-09：新增 Bridge 防火墙 oneshot systemd 单元，并让 Bridge 服务依赖该单元；隔离假 iptables 重复应用/移除验证和单元关系自检通过。当前主机没有 systemd-analyze，未在生产安装，Docker 重启恢复仍需部署后验证；`rtk git diff --check` 通过。
- OCR M-10：生产基线台账已更新为 Bridge→New API HTTPS、服务器受控配置及只读模型目录检查的当前状态，并明确实际推理、共享余额和扣费尚未验收；状态断言和 `rtk git diff --check` 通过。
- OCR M-11：生产 Compose 的日志挂载目标已从未使用的 `/app/logs` 改为 New API 工作目录 `/data` 下的默认 `./logs` 即 `<AI平台日志目录>`；Compose 解析与挂载映射断言通过，`rtk git diff --check` 通过。未部署。
- OCR M-12：核验文档的收口规则已明确标注为 2026-10-08 首轮快照，并链接到 2026-10-09 P08 后续状态；日期、状态引用断言和 `rtk git diff --check` 通过。
- OCR M-13：P08 验收范围已限于 HZC AI 关闭时的健康、凭证与只读模型目录检查；真实推理、充值、共享余额和扣费明确要求先通过 P06 并单独授权。门禁文案断言与 `rtk git diff --check` 通过。
- OCR M-14：AGENTS.md 的测试资产与后续待办回写目标已明确为 `.project-workflow/PROJECT.md`；与状态、开始、结束及发布门禁脚本的目标路径逐项比对通过，`rtk git diff --check` 通过。
- OCR M-15：备份脚本已在确认无同名文件后安装 EXIT 清理；模拟 `pg_dump` 写出部分数据后失败，临时 dump 和校验文件均未残留；bash -n 与 `rtk git diff --check` 通过。
- OCR 二轮 M-01：HTTP 与内部 HTTPS 服务现在并发排空，共用同一关闭期限但不会互相耗尽；回归测试确认两个监听器都开始关闭后才释放活动请求。定向 Go 测试通过，未跑全量回归。
- OCR 二轮 M-02：检查点已确认未跟踪，`.gitignore` 新增精确忽略规则；验证文件仍在磁盘且不出现在 Git 状态中，通过；未删除检查点。
- OCR 二轮 M-03：生产核验记录与基线台账现明确防火墙单元仅为本地候选、未在服务器安装或做 Docker 重启验证，宿主机无端口发布但访问限制待 P08 验收；状态断言与 `rtk git diff --check` 通过。
- OCR 二轮 Bridge 低风险项：Bridge HTTP Server 增加 120 秒空闲连接超时，同时保持流式请求的 ReadTimeout/WriteTimeout 不限时；无效 TLS 文件路径测试确认配置加载失败并保留底层错误。两个定向 Bridge Go 测试通过，未跑模块全量构建。
- OCR 二轮工作流状态表：已将“历史记录”的表头和分隔线移到首条数据之前；检查确认 30 条数据都在表格结构内。
- OCR 二轮状态一致性：工作流摘要与 Transition 已统一为最近一次有据服务器回读 `AiPlatform__Enabled=false`（2026-10-09），并注明本轮未复核；文案一致性断言通过。
- OCR 二轮工作流自检：`verify_workflow_setup.py` 现同时要求发布门禁脚本 `verify_workflow_gate.py` 与 AGENTS 规则标记；回归测试 2/2、自检通过，未生成 pycache。
- OCR 二轮 P05 核验记录：删除“New API 尚未启动”的过时描述，改为说明 AI 平台独立 Compose/网络及 HZC `deploy-api-1` 仍运行；日期段一致性断言通过。
- OCR 二轮 P08 门禁：防火墙未安装、Docker 重启恢复未验证已同步写入待办、当前门禁和 Transition；10 项待办数与状态断言通过。
- OCR 二轮基线隐私：生产台账不再记录运维者本机绝对路径；本机受控 SSH 配置、权限 600 与禁止回显规则保留，脱敏断言通过。
- OCR 二轮基线数据状态：台账已按 P01/P05 核验结果记录 PostgreSQL 15.19、独立 `/data` 数据盘及服务绑定目录；不存在待核验旧文案，状态断言通过。
- OCR 二轮基线凭据来源：已把 P04 未来时更新为服务器受控 `production.env` 路径，并明确不读取或记录实际秘密值；文案断言通过。
- OCR 二轮部署说明：重复且易过期的 Bridge 运行状态改为指向生产核验记录和状态 JSON；配置说明中的状态引用与重复段落断言通过。
- OCR 首轮低风险 L-03：P03 执行清单现说明根 Dockerfile 依赖下载无输出后采用同版本固定工具链离线编译、手动组装镜像，并链接详细核验记录；文案断言通过。
- OCR 二轮备份保留：轮转清理对缺失旧校验文件使用 `rm -f`；临时 Linux 工具模拟 15 个旧备份且一个缺少 sidecar，备份仍成功并保留最新 14 份；脚本语法检查通过。
- OCR 三轮高风险修复：备份脚本 release 校验现接受安全的镜像标签字符；用真实格式 `tls-c69fcdd9cab1` 完成隔离备份/轮转模拟，并确认 `../outside` 被拒绝；未触碰生产文件。
- 本轮复核回归：root 关闭/TLS 定向测试、Bridge 空闲超时/无效证书测试和工作流单测 `6/6` 通过；自检、两个 shell 脚本语法及 `rtk git diff --check` 通过。未跑全量 Go 构建或生产 systemd 验证。
- 本次后台价格编辑器修复（仅本地代码，未部署）：管理员模型定价面板停止请求受“模型广场”开关保护的公开 `/api/pricing`，沿用管理员配置接口返回的模型元数据；“模型广场”保持关闭。定向回归测试 `editor-layout.test.tsx` 9/9、前端类型检查和三个变更文件的 oxlint 均通过。生产页面未重新验证。
- OCR 定价示例提示处理：停用公开目录请求后，未配置任务插件变体的模型不显示来自公开目录的示例 JSON；管理员仍可使用 `usage_schema` 配置表达式，计费输入和结算行为不变，因此保留目录隔离并将提示缺失记录为预期的便利功能降级。
- 暂缓项：HZC AI 开关仍为 false，运行中的 HZC 容器尚未加载新项目凭证；本轮没有重启 HZC，没有发起模型推理或扣费。共享余额业务验收、真实支付/模型能力和异地备份继续按既有待办处理。

## 后续待办

- 延后事项：有，修复后 OCR 最终复审、New API TLS 证书目录生产迁移、Bridge 防火墙生产安装和 Docker 重启恢复、注册入口行为复核、HZC 实际业务调用与共享余额验收、真实余额上线所需的异地备份/隔离恢复、真实支付和合法模型能力仍待处理。
- 待办清单：见下表，共 12 项（4 项已完成，8 项待处理）。

| ID | 事项 | 暂缓原因 | 触发条件 / 计划时间 | 优先级 | 验收结果 | 关联产物 |
| --- | --- | --- | --- | --- | --- | --- |
| TODO-20261008-AIPLATFORM-SERVER-001 | 完成生产服务器盘点及 AI 域名隔离 | 本轮已完成 | 2026-10-08 | P1 | P01/P02 通过；AI 域名独立站点生效，HZC 入口保持正常 | `deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-REGISTRATION-001 | 复核线上关闭注册后的用户入口行为 | 线上 `/api/status` 已回读为 `false/false`；尚未验证登录页隐藏注册链接与注册 API 拒绝新注册 | 注册入口验收阶段 | P1 | 注册入口隐藏，注册 API 拒绝新注册 | `docs/生产服务器部署状态.json` |
| TODO-20261008-AIPLATFORM-CREDENTIAL-001 | 为 HZC 创建并安全保存项目凭证 | 本轮已完成 | 2026-10-09 | P1 | 商户 Key 只存于 Bridge 受控配置；项目凭证验证、只读 `GET /v1/models` 和内网 HTTPS readiness 均通过；HZC AI 保持关闭，未调用模型 | `docs/项目凭证管理.md`；`deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-BRIDGE-IPPORT-001 | 配置并验证 Bridge 内网域名 HTTPS 入口 | 本轮已完成 | 2026-10-08 | P1 | HZC 容器通过 `https://<BRIDGE_INTERNAL_FQDN>:3001` 完成 TLS 1.2/1.3、证书链和主机名校验；宿主机无端口映射。正式凭证与业务计费另由 CREDENTIAL TODO 跟踪 | `deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-BRIDGE-UPSTREAM-TLS-001 | 部署并验证 Bridge 到 New API 的 HTTPS 上游链路 | 本轮已完成 | 2026-10-08 | P1 | 专用证书与续期模拟通过；内网临时探针经证书链和主机名校验，在 TLS 1.2/1.3 下到达 New API；不发布宿主机端口，AI 门户及 HZC 现有功能正常。正式 Bridge 请求与凭证验收由 CREDENTIAL TODO 跟踪 | `deploy/production/生产对照核验.md` |
| TODO-20261009-AIPLATFORM-BRIDGE-FIREWALL-001 | 安装 Bridge 防火墙 systemd 单元并验证 Docker 重启恢复 | 本地规则和单元已准备；本轮未对生产主机安装或改动防火墙 | P08 生产变更验收阶段 | P1 | 安装并启用防火墙与 Bridge 单元；在生产主机校验 systemd 单元、放行/拒绝规则和 Docker 重启后的规则恢复，不改 HZC 服务或网络 | `deploy/production/bridge-firewall.sh`；`deploy/production/systemd/ai-platform-bridge-firewall.service` |
| TODO-20261008-AIPLATFORM-BACKUP-001 | 增加服务器外备份并在真实业务 schema/数据上复验恢复 | 当前同机备份仅覆盖首次启动前的空库；服务器故障可能同时影响主数据和备份 | 开放真实充值前 | P1 | 服务器外备份、保留链及业务数据隔离恢复全部通过 | `docs/生产服务器部署状态.json` |
| TODO-20261008-AIPLATFORM-PAYMENT-MODEL-001 | 验收真实支付及 HZC 共享余额业务能力 | DeepSeek 通道/价目由新待办跟踪；尚未完成共享余额业务调用、充值、支付回调或扣费验收 | 负责人明确授权业务验收且支付/上游凭据就绪后 | P1 | HZC 共享余额业务调用通过；支付回调、余额到账与退款边界，以及真实 AI 调用和扣费审计完成验收 | `docs/部署与凭据说明.md` |
| TODO-20261009-AIPLATFORM-DEEPSEEK-001 | 仅配置 DeepSeek V4 Flash 通道与官方价目 | 内置 DeepSeek 价源请求 `/api/pricing` 返回 401，官方文档未找到机器可读价格接口；不能安全地把已保存密钥用于未文档化路径。当前模型表达式为零价，且表达式引擎没有法定节假日日历 | 确认官方可用价格源及认证/节假日策略后 | P1 | 只同步 DeepSeek Flash 官方价目，保留 HZC 所需模型别名；不改其他模型或全局汇率；不发起推理或扣费；节假日不误按峰时计价 | `pkg/billingexpr/expr.md`；DeepSeek 官方价目页 |
| TODO-20261009-AIPLATFORM-PRICING-EDITOR-DEPLOY-001 | 发布并验证后台价格编辑器修复 | 本地修复已通过定向验证；生产发布门禁仍为 `NEED_VERIFY`，避免绕过项目门禁直接发布 | 全局发布门禁通过后 | P1 | “模型广场”保持关闭时，管理员仍可编辑模型价格，页面不再因 `/api/pricing` 返回 `pricing is disabled` 报错；不启用模型推理或扣费 | `web/src/features/model-pricing/model-pricing-panel.tsx`；`web/src/features/model-pricing/__tests__/editor-layout.test.tsx` |
| TODO-20261010-AIPLATFORM-TLS-CERT-MOUNT-001 | 将 New API TLS 证书源移出可写数据挂载并发布 | 本地候选已更新 Compose 与 Certbot 钩子；生产证书目录未迁移，尚未发布或重启服务 | 下一次受控 P08 发布前 | P1 | 证书源目录位于 New API 可写 `/data` 挂载之外；Certbot 写入路径与只读挂载一致；迁移后 TLS readiness 通过 | `deploy/production/docker-compose.yml`；`deploy/production/certbot/21-deploy-ai-platform-new-api-cert.sh`；`docs/部署与凭据说明.md` |
| TODO-20261010-AIPLATFORM-OCR-FINAL-REVIEW-001 | 在下一独立审查周期核对 R12 修复 | R12 12/12 项完成并补齐 R11 未完成的 3 项；2 个中优先级意见均已处理，但修复后未再经 OCR 复审。本轮已达到技能规定的三轮上限 | 下一独立审查周期复核 R12 两项修复；本轮不再启动 OCR | P1 | 核对已记录的修复和验证，并确认所选范围无未处理的高/中优先级问题 | docs/reports/2026-10-09-ocr-review.md |

## 当前门禁

- 结论：NEED_VERIFY
- 条件：R8 至 R12 已知的中优先级意见均已处理并验证；R12 已覆盖 R11 未完成的 3 项。R12 修复后未再经 OCR 复审，下一独立审查周期仍需核对。生产 New API 证书目录迁移未执行；Bridge 防火墙单元未部署且 Docker 重启后的规则恢复未验证。HZC AI 最近一次有据的服务器回读为 2026-10-09 false，本轮未复核；未进行真实推理、充值、支付、共享余额或扣费验收。
- 阻塞项：下一独立审查周期核对 R12 修复；受控发布前迁移 New API TLS 证书并验证 Compose、Certbot 钩子及 readiness；在生产安装 Bridge 防火墙单元并验证 Docker 重启后的规则恢复；DeepSeek 官方机器可读价格端点与节假日处理方式仍未确认，后台价格编辑器修复尚未部署。生产发布门禁仍为 NEED_VERIFY。未读取或发送渠道密钥，未修改生产价目。真实支付、模型推理与扣费仍按 P06/P09 门禁处理。

## 数据库证据

- db-mcp 状态：未连接；本轮未使用 `127.0.0.1:13306`。
- 数据范围与业务口径：仅为 AI 平台 PostgreSQL 创建发布前备份；没有查询、写入或恢复 AI 平台业务记录，没有访问 HZC MySQL。
- 只读证据：AI PostgreSQL 备份位于 `<AI平台备份目录>/postgres/pre-upstream-tls-20261008-c69fcdd9cab1.dump`；原 Compose 与生产环境文件备份位于 `<AI平台备份目录>/pre-upstream-tls-20261008-c69fcdd9cab1`。HZC Nginx 配置哈希与部署前一致，HZC API 健康检查为 200。
- 结构变更：New API 在其独立 PostgreSQL 内完成自身迁移；无 HZC 结构变更。

## Transition

- Flow：7. 运行与优化 / 生产部署准备 P08（R8 至 R12 已知中优先级意见均已处理；R12 修复后的 OCR 复核留待下一独立审查周期；生产 TLS 迁移、防火墙持久化与业务验收待受控验证）
- Gate：NEED_VERIFY；R12 覆盖此前未完成的 3 项并审查 12/12 项，已发现的中优先级意见已处理。因达到 OCR 技能规定的三轮上限，修复后未再审查，不能声称最终 OCR 通过。生产 New API 证书目录迁移未执行；Bridge 防火墙单元未部署，Docker 重启恢复未验证；HZC AI 最近一次有据回读为 2026-10-09 false，本轮未复查。未调用模型推理或扣费。
- Next：下一独立审查周期核对 R12 两项修复；受控发布前迁移 New API TLS 证书、验证 Compose/Certbot/readiness，安装 Bridge 防火墙单元并验证 Docker 重启恢复；之后核实 DeepSeek 官方机器可读价格与节假日规则、部署价格编辑器修复并验收。HZC AI 保持关闭；不得发起真实推理、充值、支付或扣费，直至独立门禁通过并获另行授权。

## 历史记录

| 时间 | 阶段 | 动作 | 结果 |
| --- | --- | --- | --- |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 更正 Bridge 入口记录 | 不使用 `<Bridge内网域名>`；仅将 HZC 可达的内网 IP + `3001` 作为目标，当前宿主机尚未映射端口，未做生产网络变更 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 通过隔离探针选择并验证 Bridge 网络入口 | HZC 容器访问宿主机 `<AI平台服务器内网地址>:3001` 超时；访问共享 Docker 网络地址 `<Bridge容器内网地址>:3001` readiness 返回 200；生产 Compose 不发布宿主机端口，正式凭证接入仍待完成 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 核对 HZC 生产 HTTPS 门禁 | HZC 启用 AI 时拒绝 HTTP 地址；Bridge 当前仅支持 HTTP。未改 HZC 配置或代码、未启动正式 Bridge；后续先建立可信内网 HTTPS |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 建立 Bridge 专用证书和内网 HTTPS | 专用证书已签发；Docker 内网域名别名生效；HZC 容器 TLS 1.2/1.3 readiness 验收通过；探针已清理。正式凭证和 HZC 配置仍待处理 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 准备 Bridge→New API HTTPS 上游链路 | 本地 New API TLS 监听、生产 Compose 和专用续期钩子已准备；定向测试及 Compose 校验通过；SSH 公钥认证失败，因此未在服务器签发证书、发布镜像或重建服务；生产仍需部署验收 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 按 HZC 受控连接说明恢复 SSH 并部署 Bridge→New API HTTPS | 备份 AI PostgreSQL、原 Compose 与生产环境；签发专用证书并通过续期模拟；发布 New API TLS 镜像，TLS 1.2/1.3 内网探针通过；AI 门户和 HZC 健康回读通过，HZC 配置/数据库未动。Bridge 仍 disabled；预先存在的企业 Token 字段未读取或修改，归属待核实 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 比对 HZC 与 AI Bridge 本地及线上配置 | 本地 HZC AI 保持关闭、项目 ID 为 `hedezhongcheng`、企业标识为 `platform-ops`；线上 HZC 只有 `AiPlatform__Enabled=false`。线上 AI Bridge 项目 ID 和企业映射均为 `NOT_CONFIGURED`；Bridge disabled/inactive。AI 门户与 HZC API 健康检查 200；未改生产配置、未访问 HZC 数据库、未调用模型或扣费。商户 Key 在页面遮挡显示，本地旧 POC Key 与其不匹配 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P00 | 初始化项目工作流，记录共用登录网址、内部 Bridge、DNS 和预上线备份基线 | P00 完成；进入 P01 只读盘点，真实充值前仍需异地备份 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P01 | 通过 HZC 项目受控 SSH 配置只读盘点主机资源、数据盘、Docker、Nginx、时间、监听端口和 UFW | P01 通过；未连接数据库，未改 HZC 服务；进入 P02 HTTPS 核验 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P02 | 核验 DNS、证书与续期模拟；新增 AI 独立 Nginx 站点并 reload | AI HTTP 301、HTTPS 503 占位且 TLS 有效；HZC 管理/API 检查通过，原站点配置哈希未变；进入 P03 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P03 | 完成前端、后端和 Bridge 构建核验；固定镜像与 Compose 网络边界；扫描发布配置和镜像归档 | P03 门禁通过，镜像已上传并在服务器加载；归档摘要未留存，当前记录无法复核传输一致性；未启动 AI 服务，进入 P04 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P04 | 生成服务器侧随机数据库和会话秘密并设置 root-only 权限；校验生产 Compose | 实际秘密值未回读；Compose 两种 profile 校验通过；HZC 原容器仍运行，AI 服务尚未启动；进入 P05 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P05 | 启动 AI 独立 PostgreSQL/Redis；启用每日备份 timer；完成 New API schema 备份和 tmpfs 隔离恢复 | PG/Redis healthy，备份 service success/0，40 张表恢复一致，HZC 容器仍运行；进入 P07，支付/模型后置 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P07 | 启动 New API、切换 AI 专属 Nginx 到本机反向代理并验证公网健康；保存 503 占位配置用于回滚 | AI 门户/API 200，HZC 管理/API 正常且原站点哈希不变；首个管理员待负责人自行创建 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P07 | 负责人创建首个平台管理员；AI 站点根路径改为 302 跳转 `/sign-in`，避免访客看到 New API 介绍页 | root 管理员计数为 1；公网 `/` 为 302 至 `https://<AI平台公网域名>/sign-in`，登录页和 `/api/status` 为 200；`nginx -t` 与 reload 通过；回滚配置保存在 `<AI平台备份目录>/nginx/ai-platform-before-login-redirect-20261008T082408Z.conf`；本轮未修改 HZC Nginx 配置，API 状态正常；进入 P08 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 只读比较本地与线上 New API 有效注册选项及 AI Nginx 文件 | 本地注册/密码注册为 false/false，线上为 true/true；AI Nginx 配置摘要未留存，当前记录无法复核内容一致性；公网根路径跳转一致；未写线上设置，待管理会话和写入授权后同步注册开关 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 回读公网注册开关并核对商户接入信息；负责人更正 Bridge 入口为服务器内网 IP + 端口 | 线上注册/密码注册为 false/false；负责人确认商户账号分组为 `vip`，截图确认商户 API Key 启用且令牌分组为 `vip`；密钥未读取或传递；Bridge 容器监听 `3001`，生产宿主机端口映射、readiness、充值和模型调用未验证；先前域名解析检查不再作为验收条件 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 按用户请求重载 CUA Driver MCP | 配置与 `codex mcp list` 显示 `cua-driver` 已启用且含 `--grant existing-profile`；用户已报告重启，但当前任务和原任务的工具目录仍未加载该 MCP。没有读取 Chrome 页面或密钥、没有改线上配置。下一步需在重启后的 Codex 中新建或打开任务，让工具目录重新加载后继续核验 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 管理员会话就绪后只读核验 HZC 项目凭证 | 管理员控制台及受保护的凭证列表接口已核实；接口返回 HTTP 200，`hedezhongcheng` 无记录。未显示或签发密钥、未改生产配置；本轮服务器侧 AI 开关未重新回读，最近有效证据仍是 2026-10-08 的 `false`。Bridge 配置需等项目凭证操作确认及商户 Key 核验 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 配置并启用 Bridge 商户映射 | Secret 文件脱敏核验与 Compose 校验通过；Bridge systemd enabled/active，内网 HTTPS readiness HTTP 200；只读 GET /v1/models HTTP 200。HZC AI 仍关闭，未重启 HZC、未调用模型推理或扣费。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 生产 Bridge 命名统一为 `hdzc` 并保留 `hzc` 兼容 | 首轮检查失败后自动恢复原配置；第二轮备份后将 systemd 标准 profile 切换为 `hdzc`，Compose 同时保留 `hzc`。两个 profile 均验证通过，Bridge active/readiness HTTP 200；HZC AI 仍关闭且 `deploy-api-1` 未重启。备份：`<AI平台备份目录>/hdzc-profile-20261009T045804Z-870254`。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 仅接入 DeepSeek V4 Flash 并配置官方现行价目 | 负责人报告已输入官方 API Key；本轮 CUA 连续超时，未能确认保存或继续设置价格，未读取或提交密钥，也未改生产渠道/价目。当前阻塞及待办已记录，HZC AI 开关只沿用上一轮回读、未重新验证。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 根据负责人截图定位渠道保存失败原因 | 必填模型列表为空，页面提示“请先修复高亮字段后再保存”；Chrome CUA 返回 `nodeRepl.fetch request failed`，本轮未操作或保存生产渠道/价目。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 恢复本轮工作流快照 | 首次结束校验提示缺少任务快照；按项目脚本重新建立快照并回写本轮阻塞信息。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 复核渠道创建成功但列表为空 | 负责人截图同时显示“渠道创建成功”提示和“未找到渠道”空列表；CUA Chrome 连接超时，本轮无法刷新页面或核查服务端记录；未重复创建、未读取密钥、未修改价目。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 继续配置已创建的 DeepSeek 渠道价格 | 负责人报告渠道已创建并要求继续配置；本轮 CUA 再次超时，未访问管理员价格页或修改价目。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 再次确认 DeepSeek 配置范围并检查浏览器连接 | 负责人确认仅配置 DeepSeek V4 Flash 与官方现行价格；CUA 连接超时，Chrome DevTools/Playwright 仅发现 `about:blank`，未能访问已登录管理员会话；未改生产价目。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 尝试从内置 DeepSeek 官方源导入价格 | 当前已登录管理员页面可访问；内置 DeepSeek 源的 `/api/pricing` 请求返回 401，未取得价格、未应用同步。已保存模型表达式仍为零价。未复用或发送渠道密钥；官方文档未找到机器可读价格端点，故暂停认证请求与生产改价，待确认官方数据源和节假日策略。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 修复后台模型价格编辑器误请求公开价格目录 | 本地代码已让管理员模型定价面板不请求受“模型广场”开关保护的 `/api/pricing`；定向测试 9/9、类型检查和变更文件 lint 通过。未改公开目录开关或生产配置，尚未部署。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 核验后台价格编辑器修复的发布门禁 | `verify_workflow_gate.py --mode release` 返回 NEED_VERIFY，因为“当前门禁-结论”尚未 PASS；不绕过门禁发布。本地修复待后续发布并做线上验收。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 审查并修复主工作区生产配置候选 | 首轮 OCR 40/40 完成，4 条中优先级问题已处置，9 条低优先级未修；修复后 OCR 复审的 40 项全部因供应商 HTTP 402 失败，须恢复 OCR 路由后重跑。工作流单测、Compose 两个 profile、JSON、自检和差异检查通过；New API TLS 生产证书迁移与 Bridge 防火墙安装未做，生产门禁仍为 NEED_VERIFY；未提交、推送或部署。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 完成 R8/R9 中优先级问题修复 | R8 的 10 项与 R9 的 5 个独立中优先级问题均已修复并逐项验证；生产门禁仍为 NEED_VERIFY。R10 复审结果另见下行。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | R10 最终复审发现补充问题 | 43 个审查项中返回 6 条评论（3 中、3 低）；另有 2 组请求超时、1 组取消，最终复审未完成。三个中优先级问题均已修复并定向验证；未部署、未重启服务。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | R11 修复后复审 | 最终预览 45 项，42 项完成并返回 4 个中优先级问题，3 项失败；4 个中优先级问题均已修复并验证。未视为完整通过；生产未访问、未改动。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | R12 补充复审与修复 | 12/12 项复审完成，发现 2 个中优先级意见且无高优先级意见；R11 未完成的 3 项已由本轮覆盖。新增防火墙子网唯一性断言并收紧 fast 通道触发前缀；工作流单测 34/34、Bridge 配置 18/18、前端 5/5、typecheck 与规则/差异检查通过。达到 OCR 三轮上限，修复后不再启动 OCR；门禁仍为 NEED_VERIFY。 |
