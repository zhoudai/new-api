# 项目工作流状态

## 项目

- 名称：ai-platform
- 目标：在 HDZC 当前生产服务器上独立部署 AI 平台，让平台管理员和 HDZC 商家共用一个登录网址，并让 HDZC 后台用户共享商家 AI 余额。
- 负责人：周到
- 当前阶段：7. 运行与优化 / debug-flow / HZC 与 AI 平台联合排障

## 当前任务状态（2026-10-15 HZC 与 AI 平台联合排障）

- 关联 Bug：HZC `BUG-20261015-AI-001`。链路为 HZC .NET API → 本项目 Bridge → New API DeepSeek adaptor → DeepSeek 原厂。
- 当前证据：用户最新请求 `b0a8f0f4ed084ee6a79dd0e4a16844fe` 于北京时间 10:37:26 进入平台，HZC 于 10:37:56 超时；原厂官方故障覆盖该时段。单次经授权的关闭思考合成验证也在 30 秒超时，该候选已从 HZC 撤回。
- 切片 1：修复普通 JSON/表单上游请求未继承客户端取消。已复现两个入口在取消后仍访问模拟上游；修正后返回不可重试的 HTTP 408，复用现有失败退款路径，并避免自动禁用渠道。
- 测试资产-本次变更、最小验证：`relay/channel/api_request_test.go` 覆盖 JSON/表单取消后的零上游调用、408 与不可重试标记；原代码两例失败，修正后 `go test ./relay/channel -run Cancellation -count=1 -timeout=30s` 通过。
- 切片 2：修复聊天响应缺少 `choices`、或上游 `error` 缺少 `type` 时被当作成功并补估用量的问题；改为 HTTP 502 并返回失败，复用既有失败结算路径。合法工具调用的无正文响应保持有效。读取响应正文时的客户端取消返回不可重试的 408。
- 联合证据：最新用户请求最终在约 901 秒结束，单次合成验证在 900 秒结束，输出 token 均为 0。生产未保留原始响应正文，不能仅凭该日志断定实际响应属于哪一种无效格式；上述两种无效格式已在模拟上游中独立复现。
- 测试资产-联合回归：固定 Go 1.26.1 的 `go test ./relay/channel ./relay/channel/openai ./bridge -count=1 -timeout=90s` 全部通过，覆盖取消前零上游调用、运行中连接取消、响应正文读取取消、错误无 type、缺少 choices 和合法工具调用；HZC `AdminOperationAnalysisTests` 8/8 通过。两个缺陷的原实现均先观察到定向失败。
- 文档：按 asd-ste100 仅复核本次新增中文段落，更新既有 `docs/项目统一接入协议-v1.md` 的取消及失败语义；无数据库结构、余额、价目、凭证或配置项变更。
- 生产边界：本轮仅在 HZC 用户已确认后执行 1 次合成推理，未再次调用真实模型、未部署、未修改资金或价目。生产只有 1 个启用的 DeepSeek 渠道；只读渠道元数据查询不读取密钥、余额或账单。
- 候选构件：`/tmp/ai-platform-chainfix.pulEyq/new-api`，Go 1.26.1、CGO=0、linux/amd64、greenteagc 构建通过；SHA-256 `4f18be337fc6b137fbca05de6d65a00e4700281a9a0f5ae8b48dc722a95deeeb`。同目录保存仅含本轮改动的补丁与 manifest。前端源码与已部署基线无差异，嵌入资源从当前同版本源码重建并构建通过；候选入口资源哈希与生产不同，不声称前端资源逐字节一致。
- 发布确认：负责人选择暂不发布、仅保留本地修正；回滚沿用现生产镜像 `ai-platform-new-api:pricing-editor-20261010-cdc9e7c82-r1`。生产配置和容器未改动；后续发布须按生产清单完成备份与隔离恢复并另行确认。
- Git：本轮平台修复、接入协议、工作流记录和诊断现场均按职责拆分原子提交；负责人已授权统一推送至个人 fork，生产候选未发布。

## 当前门禁

- 结论：NEED_CONFIRM
- 条件：两处本地修正和固定 Go 1.26.1 三包回归、HZC 8/8 定向测试已通过；生产发布未授权，真实恢复验收尚未完成。原厂故障仍属外部阻塞。

## Transition

- Flow：7. 运行与优化 / debug-flow / systematic-debugging → bug-tracker。
- Gate：NEED_CONFIRM / NEED_VERIFY
- Next：待 DeepSeek 服务恢复或负责人重新授权时，再按生产清单评估候选发布；真实推理回归仍需单独授权。

## 当前任务状态（2026-10-10 生产部署配置审查与定向修复）

- 用户确认平台管理员与 HDZC 商家共用 `https://<AI平台公网域名>`；平台按账号权限显示不同功能。
- HDZC 后台用户通过内部 Bridge 调用 AI，共用一个商家企业计费账号和余额。Bridge 不作为公开登录入口。
- 两个项目在 `<AI平台公网地址>` 上独立运行；不复用 HDZC 应用容器、配置、MySQL 或数据目录。本轮不使用 `edu-training`。
- P00、P01、P02 已完成。2026-10-08 两个权威 DNS 均回读到 `<AI平台公网域名> A <AI平台公网地址>`；证书续期模拟通过，并已启用 AI 专属 Nginx 配置。应用未发布时 AI 域名返回 503，占位状态与 HDZC 管理端隔离；证据见 `deploy/production/生产对照核验.md`。
- P03 已完成：前端生产构建、固定 Go 1.26.1 的后端交叉编译、Bridge 定向测试、Compose 默认与当时的 HDZC profile 配置校验及 gitleaks 秘密扫描通过；镜像归档上传并加载；摘要未留存，当前记录无法复核传输一致性；镜像当时尚未启动。根 Dockerfile 完整 BuildKit 构建在下载依赖时无输出并中止，改用同版本固定工具链离线编译并组装运行时镜像；详细证据和边界见 `deploy/production/生产对照核验.md`。
- P04 已完成：服务器 `<AI平台数据目录>/production.env` 由服务器生成 PostgreSQL、Redis 和会话随机秘密，文件权限为 `600 root:root`；只回读键名/是否设置，未读取或输出秘密值。Bridge 配置在 P08 完成商户映射配置，具体验收见下文。
- P05 已完成：独立 PostgreSQL 15.19、Redis 8.10.1 在 `/data` 健康运行；每日备份 timer 已启用、手动 service 退出码 0；New API 迁移后备份恢复到 tmpfs 隔离 PostgreSQL 成功，源与目标均为 40 张 public 表。备份与主库暂在同一服务器，真实充值前仍需异地备份和业务数据恢复验证。
- P07 已完成：New API、PostgreSQL、Redis 均 healthy；首个平台管理员已创建并只读核对到 1 个 root 管理员。AI 域名根路径现由独立 Nginx 直接跳转到 `/sign-in`；登录页和 API 健康接口公网返回 200，HDZC 原入口仍通过。生产 AI 开关为 true，Bridge 已运行。首次合成请求因商户余额为零在 New API 预扣费阶段被拒绝；负责人增加余额后，一条受限合成请求成功并返回 8 个输入、1 个输出 token。后续只读对账确认该请求扣除 1 个额度点；支付未启用。
- P08 注册设置最近一次只读回读为 `false/false`；本轮发布 TLS 后未重新读取这两项，也未提交注册请求或检查注册 API 的拒绝结果。
- 商户接入准备：此前项目记录将商户账号组记为 `vip`，截图确认 API Key 令牌组为 `vip`。本轮按生产消费记录反查到关联用户 `users.group=default`、请求组 `vip`；充值倍率以账户组为准，因此商户账号 VIP 归属需后续核实，不读取或记录凭证值。
- P08 内网 HTTPS 已完成并验证：Bridge 使用独立的 Let’s Encrypt ECDSA 证书（`<BRIDGE_INTERNAL_FQDN>`，有效期至 `<证书有效期已核验>`），不共享 AI 门户的通配符私钥；Certbot DNS-01 临时 TXT 已清理，权威 DNS 没有该域名的 A/AAAA 记录。生产 Compose 将 `<BRIDGE_INTERNAL_FQDN>` 作为 `<HDZC_DOCKER_NETWORK>` 网络别名指向 `<Bridge容器内网地址>`，证书目录只读挂载，Bridge 最低 TLS 版本为 1.2，宿主机不发布 `3001`。
- 从 HDZC `deploy-api-1` 容器通过域名、证书链及主机名校验访问 `https://<BRIDGE_INTERNAL_FQDN>:3001/health/ready`；TLS 1.2 和 1.3 均返回 200。临时探针已清理。自动续期钩子已安装并验证初次复制，但未等待续期事件或对该新证书执行续期 dry-run。
- P08 Bridge→New API 内网 HTTPS 已部署：为 `<NEW_API_INTERNAL_FQDN>` 单独签发 ECDSA 证书，Certbot DNS-01 续期模拟通过；New API 使用 `ai-platform-new-api:tls-c69fcdd9cab1`，健康检查和内网 TLS 1.2/1.3 探针通过。证书续期钩子已安装；端口 `3001` 未发布到宿主机。
- 2026-10-10 已按受控连接说明迁移 New API TLS 证书目录：新目录位于 New API 可写数据挂载之外，目录权限为 `0750 root:root`，证书和私钥为 `0440 root:root`；Compose 只读挂载与 Certbot 钩子目标已同步更新。迁移前保留了仅 root 可访问的回滚备份，只重建 New API 容器并沿用原镜像；重建后 New API 健康检查为 `healthy`，内网 TLS 1.2/1.3、AI 门户 `/api/status` 和 HDZC readiness 均为 HTTP 200，Bridge、PostgreSQL、Redis、HDZC 容器未重启。Certbot timer 状态为 `active`；随后使用 systemd 管理的 Certbot 3.3.0 虚拟环境完成续期模拟，未更新生产证书或运行部署钩子。
- TLS 上游发布步骤开始前已备份 AI 平台 PostgreSQL、原 Compose 与生产环境文件；当次比对确认只有 `AI_PLATFORM_RELEASE` 变化，未改 HDZC Nginx、服务、配置或数据库。AI 门户与 HDZC API 仍健康。
- 2026-10-09 在已打开的管理员浏览器会话中确认此前 `hedezhongcheng` 无项目凭证，随后签发并轮换到第 2 版；项目凭证写入 HDZC 服务端 `<HDZC配置目录>/production.env`，项目凭证校验返回 HTTP 204。凭证值未显示。服务器回读确认 `AiPlatform__Enabled=false`；未重启 HDZC，运行中的 HDZC 容器尚未加载新项目凭证。
- 商户 New API Key 已由负责人保存到 `<AI平台数据目录>/production.env`。脱敏回读确认项目 ID 正确、`hdzc-shared` 映射存在且唯一、文件权限为 `0600 root:root`；未读取或输出密钥。Bridge systemd 已启用并运行，容器在内网 HTTPS readiness 返回 HTTP 200；通过 Bridge 发起只读 `GET /v1/models` 返回 HTTP 200，验证项目凭证与商户映射可用。该请求只取模型目录，不调用模型；HDZC AI 仍为 false，HDZC 服务未重启。
- 本次配置前将 AI 平台和 HDZC 的生产环境文件、Bridge Compose 与 systemd 单元备份到 `<AI平台备份目录>/hzc-bridge-20261009T031407Z`；备份文件权限为 0600。未启动 Bridge、未调用模型或执行扣费。
- 2026-10-09 按负责人确认，将 HDZC A-02 管理端运营分析使用的固定企业标识从 `platform-ops` 改为 `hdzc-shared`；各招聘企业仍使用各自企业标识。同步更新 HDZC 服务端代码、默认配置、测试和配置说明，以及本地 HDZC 配置、AI 平台说明与 SSO 测试。本地 HDZC 开发配置也已设为 `AiPlatform__Enabled=false`。生产 `<HDZC配置目录>/production.env` 已在备份 `<AI平台备份目录>/hzc-enterprise-id-20261009T041551Z/hzc.production.env` 后写入新标识；脱敏回读为 `AiPlatform__EnterpriseId=hdzc-shared`、`AiPlatform__Enabled=false`，权限保持 `640`。未重启 HDZC，`deploy-api-1` 仍在运行，进程尚未加载新配置。
- 本次命名子任务：生产 Compose 与 systemd 已统一使用标准 Bridge profile `hdzc`，旧 `hzc` profile 保留为兼容别名；内部 Compose 网络键为 `hdzc-internal`，底层 Docker 网络 `<HDZC_DOCKER_NETWORK>`、网段、地址、端口和 Bridge 行为保持不变。systemd 已重载并仅重启 Bridge，备份位于 `<AI平台备份目录>/hdzc-profile-20261009T045804Z-870254`。
- 2026-10-09 当前接入请求：负责人希望只配置 DeepSeek V4 Flash 并按官方现行价格计费。管理员页面已读到 `deepseek-v4-flash` 的已保存表达式为零价。上游价格同步选择内置 DeepSeek 源后，请求 `/api/pricing` 返回 `401 Authorization Required`，没有可应用的价格；同步请求未复用渠道密钥，未读取、复制或发送密钥，也未修改价目。官方文档列出 `/models` 模型列表接口，未找到机器可读价格接口。不得将已保存密钥发送到未文档化的价格路径；自动导入需先确认可用的官方价格源及安全认证方式。
- 后台价格编辑器修复已在生产部署并验收：管理员面板不再依赖公开价格目录，任务示例改由后台模型元数据提供。管理员浏览器中 `deepseek-v4-flash` 编辑窗口正常打开，无 `pricing is disabled` 提示；没有保存价目、发起模型调用或扣费。
- 预上线备份暂放服务器数据盘上的独立目录；服务器或数据盘故障可能同时影响主数据和备份。真实充值开放前必须增加异地备份并完成恢复验证。真实支付未启用，HDZC 真实业务模型调用仍待验收。
- 本轮通过受控 SSH 配置 AI 独立 Nginx 站点、镜像、秘密文件、PostgreSQL/Redis、备份定时任务并发布 New API；随后仅调整 AI 站点的 `/` 路径，使其跳转到 `/sign-in`。该阶段 HDZC 站点配置、服务与数据库未改动，`deploy-api-1` 保持 Up。

## 测试资产

- 本次变更：按 `deploy/production/生产基线台账.md` 恢复受控 SSH 连接成功；只读 `docker ps` 确认 HDZC `deploy-api-1` 与 AI 平台 New API、Bridge、PostgreSQL、Redis 容器运行。读取商家账号组和 `TopupGroupRatio.vip` 需要查询 AI 平台生产 PostgreSQL；本轮查询因生产数据库读取审批被拦截，尚未读取数据。负责人确认 HDZC 商家账号已改为 `vip`；历史快照（2026-10-10）记录账号组为 `default`、`TopupGroupRatio.vip=0.67`、`GroupRatio.vip=1`，不能据此判断当前线上值。随后指定的 HDZC 连接说明只定义 HDZC MySQL（本机 `127.0.0.1:13306`）连接方式，与待查的 AI 平台 PostgreSQL 不同；未使用该连接说明尝试访问，也未绕过数据库读取审批。未更改生产配置或数据库，未发起模型请求。新增 `docs/AI平台计费配置与计算规则.md`，汇总生产只读快照中的 DeepSeek 价格、CNY/USD 汇率、额度点、分组倍率、人工充值规则和在线充值公式；明确官方人民币价与美元价乘平台汇率 7 的差异。同步生产 TLS、Certbot、价格编辑器和 Bridge 防火墙状态，以及 OCR R12 独立复核记录。10 个工作区变更文件按生产核验、计费文档、OCR 复核和工作流记录拆成 4 个原子提交后统一推送；未改生产配置或支付。
- 最小验证：`python3 -m unittest discover -s .project-workflow/tests -p 'test_bridge_deployment_config.py'` 18/18 通过；生产状态 JSON 解析通过；`git diff --check` 通过。未发起模型请求、扣费或生产写入。
- OCR 三轮问题处理顺序：P0 备份标签校验拒绝实际发布标签（已修并验证）；P1 Bridge 双网卡暴露/防火墙失配、生产明细进入公开仓库、备份并发互相覆盖、工作流快照解析与清理（均已修并定向验证）；P2 发布门禁步骤、TLS 可选行为、商户映射示例、历史表格格式和 Python 缓存忽略（均已修并核验）。定价示例提示缺失属于关闭公开模型广场后的非计费功能变化；Go 标准库默认已有 1 MiB 请求头限制，不重复设置。
- 本次变更：New API 增加可选的内网 HTTPS 监听（TLS 最低 1.2）；生产 Compose 使用 `<NEW_API_INTERNAL_FQDN>:3001` 作为 Bridge 上游、增加 Docker 内网别名和只读证书挂载；新增 Certbot 专用证书部署钩子。HDZC A-02 固定企业标识已更新为 `hdzc-shared`。Bridge 商户映射、内网 HTTPS 与命名已部署。2026-10-10 Docker 重启后从运行容器回读到 `AiPlatform__Enabled=true`；未发起模型请求。DeepSeek 生产价目与官方当前数值匹配；负责人决定本轮忽略节假日差异，不调整计价表达式。官方机器可读价源未确认，价格变化按官方页面人工复核。后台价格编辑器修复通过管理员面板隔离公开价目请求，任务示例由后台模型元数据提供；本次改动不触碰计费表达式、价格数据或数据库结构。OCR 候选改动和未改生产服务器的审查记录见下文。本次后台价格编辑器修复已按通过的发布门禁部署，生产 UI 验收通过；未改价目或数据库。
- 2026-10-10 DeepSeek 价目只读核验：官方文档将 `deepseek-flash` 列为当前模型名，旧 `deepseek-v4-flash` 暂时路由到 V4.1-Flash；官方明确中国法定节假日全天按闲时价。生产渠道仅配置 `deepseek-v4-flash`，默认组与 `vip` 组均启用；生产 `vip` 分组倍率为 1。数据库只读回读到 `tiered_expr`：缓存未命中/命中/输出峰时分别为 USD $0.30/$0.006/$1.20 每百万 tokens，闲时为一半，与官方现行数值一致；表达式仅按 UTC 星期和小时分峰，缺少中国法定节假日排除，因此节假日落在峰时窗口会按官方闲时价的两倍计费；负责人决定本轮接受这一差异，不改表达式。官方价目页建议定期查看，查阅的官方文档没有机器可读价格 API；此前 `/api/pricing` 返回 401。SQL 在只读事务内执行，未读取密钥字段、未改数据库或生产配置、未发起推理或扣费。
- 本次生产证书迁移验证：服务器 Compose 校验及 Certbot 钩子 `sh -n` 通过；新目录、文件权限、只读挂载和钩子目标一致；New API 沿用原镜像重建后健康检查为 `healthy`，TLS 1.2/1.3 `/api/status`、公网 AI 门户 `/api/status`、HDZC readiness 均为 HTTP 200；Bridge、PostgreSQL、Redis 和 HDZC 容器 ID 未变。Certbot timer 状态为 `active`；迁移后 Certbot 3.3.0 续期模拟成功，未更新生产证书或运行部署钩子。
- 本轮生产防火墙与 Docker 重启验证：先保存生产 Compose、systemd 单元和原专用链规则；将 Bridge 监听从所有容器网卡收窄到 HDZC 网络固定地址，宿主机仍不发布端口。防火墙与 Bridge systemd 单元均 enabled，`systemd-analyze verify`、生产 Compose 校验及 `iptables -C` 规则核对通过。Docker 服务重启后，防火墙、Bridge 及原运行的 5 个容器恢复；New API/PostgreSQL/Redis 为 healthy，Bridge 内网 readiness 与公网 HDZC API readiness 均 HTTP 200。HDZC 网络 TCP 探测通过；其他网络的探测被 Docker 网络隔离阻断，但防火墙 DROP 计数未增加，因此本轮没有独立观测到 DROP 规则命中。运行容器的 `AiPlatform__Enabled=true`；未发起推理、充值、支付或扣费。
- R8 的 10 项与 R9 的 5 个独立中优先级问题均已修复并逐项验证；R10 的 3 个、R11 的 4 个中优先级问题及 R12 的 2 个中优先级意见均已处理并完成对应验证。R12 覆盖此前 R11 未完成的 3 项；R12 两项修复已在独立审查周期复核，所选 2 个文件均完成且无评论；该范围不代表整个仓库或生产候选审查通过。
- R12 修复独立复核：OCR 会话 `76d043be-9749-4756-ae5b-74fbe75e4a1b`，选中 `AGENTS.md` 与 `.project-workflow/tests/test_bridge_deployment_config.py`，2/2 完成，0 条评论、0 失败；使用 DeepSeek 官方 API 直连，耗时 1 分 16 秒。复核前 `ocr-rules --check` 确认规则最新，并通过一次性规则副本移除这两个路径的重复问题抑制；未改代码、未运行代码测试。
- 最小验证：`CGO_ENABLED=0 rtk test go test . -run TestNewInternalTLSServer -count=1`、`rtk test go test ./bridge`、Certbot 部署钩子 `sh -n`、`rtk git diff --check` 均通过；Go 1.26.1 离线构建 linux/amd64 镜像成功。Certbot DNS-01 续期模拟通过；内网临时探针在 TLS 1.2、1.3 下均通过证书链、主机名和 `/api/status` 检查。生产 New API 健康检查为 `healthy`，宿主机仅映射 `127.0.0.1:3000`；AI 门户/API 为 200，HDZC 管理入口为 302、HDZC API 为 200。2026-10-09 项目凭证校验返回 HTTP 204；脱敏回读确认 HDZC AI 为 false、企业标识为 `hdzc-shared`，未重启 HDZC。商户映射仅确认存在且唯一、未读取密钥；文件权限 `0600 root:root`。生产 Compose quiet validation 通过；Bridge systemd enabled/active、容器运行，内网 HTTPS readiness HTTP 200；只读 `GET /v1/models` 返回 HTTP 200 且响应有效。未调用模型、扣费或重启 HDZC。HDZC A-02 定向 .NET 测试 `8/8`、API build `0` warning / `0` error；AI 平台普通本机 Go 测试因缺少系统 C 头文件失败，`CGO_ENABLED=0` 定向测试通过。安全依据：OWASP ASVS 5.0.0 V12.1.1、Web Service Security Cheat Sheet、Transport Layer Security Cheat Sheet；不代表完整 ASVS 验收。证据见 `deploy/production/生产对照核验.md`。 第三轮定向验证：工作流单测 9/9、前端 typecheck 与目标文件 oxlint 通过；Compose JSON 与 Bridge 监听/防火墙地址一致；备份脚本语法、标签/路径拒绝、缺失输入诊断通过；状态 JSON 和文档脱敏断言通过。Go 官方文档确认 MaxHeaderBytes 为 0 时使用 1 MiB 默认值。systemd-analyze 未安装。第四轮 OCR 选中 39 个文件，29 个完成、10 个因 LLM 请求超时失败，共 8 条评论，已全部处理：Bridge TLS 复用预加载证书，`rtk test go test ./bridge` 通过；发布门禁拒绝“未执行”占位，全量工作流回归 12/12 通过。此轮 OCR 仍有 10 个文件因 LLM 请求超时未完成，需补审后才能判定完整通过；四个工作流入口共用向上查找状态文件的解析器，嵌套目录下状态、提交门禁和开始快照验证通过；工作流脚本回归测试 12/12 通过。提交钩子安装器遵循 `core.hooksPath`，钩子运行时解析仓库根目录，shell 语法检查和回归测试 4/4 通过；部署说明已更正固定地址绑定已生效、防火墙持久化单元未部署，交叉状态断言通过；生产台账已移除摘要未留存时的哈希一致性主张，JSON 解析与摘要声明断言通过；Docker 忽略规则覆盖 `.env*` 与 `*.env`，保留 Dockerfile.local；Git 忽略环境变量文件并放行模板，模式断言和 `docker compose config --quiet` 通过。补审首批 4/24 个文件新增 3 条意见：Docker 忽略规则已补 `*.env`；Bridge 改为通过 `GetCertificate` 复用同一证书指针，`go test ./bridge` 与 `go vet ./bridge` 通过。请求头上限意见未改代码：Go 官方 `net/http` 文档明确默认限制为 1 MiB，显式写入同值不改变行为。 价格编辑器定向测试、前端全量 `bun run test`、`bun run typecheck` 和 `bun run build` 均通过；linux/amd64 候选镜像构建成功；Certbot 3.3.0 续期模拟、生产 Compose 校验和 root-only 回滚配置备份通过。部署后 New API healthy，本机与公网 `/api/status` 均返回 200；Bridge、PostgreSQL、Redis 容器 ID 未变。管理员浏览器实际打开 `deepseek-v4-flash` 编辑器，表单和价格预览正常；测试断言管理端不请求 `/api/pricing`。
  - R8 工作流测试 26/26、Bridge Go 测试与 vet、Shell 语法、Compose 双 profile 配置及 diff 检查通过。R9-M-01 的 Bridge Go 测试与 vet 通过；生产 Nginx/systemd/容器运行未验证。
  - R9-M-01 定向验证：rtk test go test ./bridge 与 rtk proxy go vet ./bridge 均通过；未做生产慢速请求探测。
  - R9-M-02 定向验证：Bridge 部署配置测试 13/13 通过；HDZC 与 edu-training Compose config --quiet 均通过。未生成证书或启动容器。
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
- 本次生产限量验收与计费对账：请求 `prod-smoke-balance-20261010-01` 的 New API 消费记录为输入 8、输出 1、缓存 0、`off_peak`，扣除 1 个额度点；账户累计使用 1 点、请求 1 次、剩余 357142 点。生产 `USDExchangeRate=7`、`Price=7`、`vip` 调用倍率 1；当前 CNY 余额与截图的 ¥5 相符。未运行代码测试；未查询 DeepSeek 官方账户实际账单、未验收 HDZC 真实业务流程。
- 本轮增量审查与验证：OCR 会话 `1683ad3b-f1fc-4c60-99a9-ba964f9d1aba` 完成主工作区 40/40 审查项，13 条评论（4 中、9 低；低优先级未修）；3 个 LLM 请求错误已记录，不影响审查项完成状态。处置防火墙依赖顺序、Bridge systemd 恢复说明、P08 状态账本和 New API TLS 证书挂载问题。修复后 OCR 复审会话 `b0895c82-85eb-4506-b7ac-3e92980038ca` 的 40 个审查项全部因请求 HTTP 402 失败，不能当作零问题结果；恢复服务后仍须复审。工作流单测 17/17、`hdzc`/`hzc` Compose 配置、状态 JSON 解析、自检、结束校验、Certbot 脚本语法和 `rtk git diff --check` 通过。生产证书目录迁移、Bridge 防火墙安装和 Docker 重启恢复仍未执行。未提交、推送、部署、重启 HDZC、调用模型或扣费。
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
- 本次配置命名验证：服务器候选 Compose 的 `hdzc` 与兼容 `hzc` profile 均通过 `docker compose config --quiet`；本机未安装 `systemd-analyze`，未执行 `systemd-analyze verify`。部署后两个 profile 均解析到同一 Bridge 容器，systemd 使用 `hdzc`，Bridge readiness HTTP 200。HDZC AI 仍为 false，HDZC 容器 ID 与启动时间未变；工作流结束校验通过。
- 全量回归：本次发布范围的前端全量测试 `bun run test` 通过，前端生产构建 `bun run build` 通过；未运行 Go 全量测试或 HDZC 真实模型、共享余额及扣费业务验收，这些不属于价格编辑器改动路径。
- OCR 补审收口：第三轮会话 `869f5b39-1b0b-4528-8a49-1ee1e9879e91` 选中 39 个文件、18 条评论、全部文件完成（1 个 LLM 错误）；第四轮 `787f87e5-3a72-472b-90bc-23c825e10fe1` 选中 39 个文件、29 个完成、10 个请求超时、8 条评论；补审 `0caafb2a-9ea4-4575-b9ab-e8c57dfbba71` 选中 24 个文件、全部完成、8 条评论。补审发现并修复发布门禁可被“未执行；说明文字”绕过的问题，`read_field` 和状态路径改由共享模块维护；部署文档区分 Bridge 服务已安装与防火墙持久化单元未安装，2026-10-08 状态表标注为已被 2026-10-09 记录取代。补审中重复建议显式设置 1 MiB `MaxHeaderBytes` 未采纳：Go `net/http` 在零值时已有相同默认上限。工作流脚本回归测试 12/12、自检、文档状态交叉断言及 `git diff --check` 均通过；第四轮 10 个超时文件已在 24 文件补审中完成，第五轮 9 文件和第六轮 2 文件定向复审均完成，无高/中优先级问题；状态文档占位符一致性断言通过。低优先级按 ocr-review-loop 规则不进修复队列；报告仅保留高/中问题，已完成项用 ✅ 标记。
- DeepSeek 配置验证：管理员浏览器已连接；内置 DeepSeek 上游价格源返回 HTTP 401，价格列表 0 项且“应用同步”禁用。读取到 `deepseek-v4-flash` 已保存表达式 `tier("base", p * 0 + c * 0)`；未修改价目、未调用模型、未产生扣费，也未读取或发送渠道密钥。自动导入仍需解决官方价格源和节假日分时规则。
- OCR H-01 本地修复：Bridge 防火墙规则改为匹配容器地址 <Bridge容器内网地址>:3001。隔离假 iptables 调用验证了 HDZC 网段放行与其他来源拒绝规则；bash -n 与 rtk git diff --check 通过。未在服务器应用规则；真实转发链路仍需服务器侧验证。
- OCR M-01 本地修复：工作流自检器现从目标项目的 .project-workflow/scripts 检查必需脚本；新增临时目标仓库回归测试。unittest 1/1 与本仓库自检通过，rtk git diff --check 通过；待负责人确认后提交。
- OCR M-02：部署说明已更新 Bridge→New API 的 HTTPS 当前状态、证书配置和业务验收边界。基于生产核验记录检查文案与状态一致，M-02 文档断言及 rtk git diff --check 通过。
- OCR M-03：部署说明已更正 Bridge systemd 当前为 enabled/active、readiness 通过，并明确 HDZC AI 仍关闭及业务验收边界。文档断言和 rtk git diff --check 通过。
- OCR M-04：内部 TLS HTTP Server 设置 10 秒请求头超时和 120 秒空闲超时；请求头大小沿用 Go 标准库默认上限，不再写入无效的默认值覆盖。测试覆盖默认/自定义地址、两种不完整证书配置及证书加载错误保留；定向 Go 测试、gofmt 与 `rtk git diff --check` 通过。
- OCR M-05：注册状态记录已区分较早 true 与同日后续 false 回读，并注明 TLS 发布后未复核及未验证注册拒绝行为。JSON 解析、证据顺序断言和 rtk git diff --check 通过。
- OCR M-06：Git hook 安装器现在也会为已有门禁和追加后 hook 设置执行权限。临时 Git 仓库回归测试 2/2、bash -n 与 rtk git diff --check 通过。
- OCR M-07：Git hook 安装器现遇到 exit 标记时拒绝自动追加，避免条件/内联提前退出绕过门禁；回归测试覆盖条件内联退出。bash -n、测试 3/3 与 rtk git diff --check 通过。该保守检测也可能匹配注释中的 exit。
- OCR M-08：已新增工作流状态初始化模板；定向测试验证模板创建、项目名替换和已有状态不被覆盖。测试 1/1、脚本 --help 与 rtk git diff --check 通过。
- OCR M-09：新增 Bridge 防火墙 oneshot systemd 单元，并让 Bridge 服务依赖该单元；隔离假 iptables 重复应用/移除验证和单元关系自检通过。当前主机没有 systemd-analyze，未在生产安装，Docker 重启恢复仍需部署后验证；`rtk git diff --check` 通过。
- OCR M-10：生产基线台账已更新为 Bridge→New API HTTPS、服务器受控配置及只读模型目录检查的当前状态，并明确实际推理、共享余额和扣费尚未验收；状态断言和 `rtk git diff --check` 通过。
- OCR M-11：生产 Compose 的日志挂载目标已从未使用的 `/app/logs` 改为 New API 工作目录 `/data` 下的默认 `./logs` 即 `<AI平台日志目录>`；Compose 解析与挂载映射断言通过，`rtk git diff --check` 通过。未部署。
- OCR M-12：核验文档的收口规则已明确标注为 2026-10-08 首轮快照，并链接到 2026-10-09 P08 后续状态；日期、状态引用断言和 `rtk git diff --check` 通过。
- OCR M-13：P08 验收范围已限于 HDZC AI 关闭时的健康、凭证与只读模型目录检查；真实推理、充值、共享余额和扣费明确要求先通过 P06 并单独授权。门禁文案断言与 `rtk git diff --check` 通过。
- OCR M-14：AGENTS.md 的测试资产与后续待办回写目标已明确为 `.project-workflow/PROJECT.md`；与状态、开始、结束及发布门禁脚本的目标路径逐项比对通过，`rtk git diff --check` 通过。
- OCR M-15：备份脚本已在确认无同名文件后安装 EXIT 清理；模拟 `pg_dump` 写出部分数据后失败，临时 dump 和校验文件均未残留；bash -n 与 `rtk git diff --check` 通过。
- OCR 二轮 M-01：HTTP 与内部 HTTPS 服务现在并发排空，共用同一关闭期限但不会互相耗尽；回归测试确认两个监听器都开始关闭后才释放活动请求。定向 Go 测试通过，未跑全量回归。
- OCR 二轮 M-02：检查点已确认未跟踪，`.gitignore` 新增精确忽略规则；验证文件仍在磁盘且不出现在 Git 状态中，通过；未删除检查点。
- OCR 二轮 M-03：生产核验记录与基线台账现明确防火墙单元仅为本地候选、未在服务器安装或做 Docker 重启验证，宿主机无端口发布但访问限制待 P08 验收；状态断言与 `rtk git diff --check` 通过。
- OCR 二轮 Bridge 低风险项：Bridge HTTP Server 增加 120 秒空闲连接超时，同时保持流式请求的 ReadTimeout/WriteTimeout 不限时；无效 TLS 文件路径测试确认配置加载失败并保留底层错误。两个定向 Bridge Go 测试通过，未跑模块全量构建。
- OCR 二轮工作流状态表：已将“历史记录”的表头和分隔线移到首条数据之前；检查确认 30 条数据都在表格结构内。
- OCR 二轮状态一致性：工作流摘要与 Transition 已统一为最近一次有据服务器回读 `AiPlatform__Enabled=false`（2026-10-09），并注明本轮未复核；文案一致性断言通过。
- OCR 二轮工作流自检：`verify_workflow_setup.py` 现同时要求发布门禁脚本 `verify_workflow_gate.py` 与 AGENTS 规则标记；回归测试 2/2、自检通过，未生成 pycache。
- OCR 二轮 P05 核验记录：删除“New API 尚未启动”的过时描述，改为说明 AI 平台独立 Compose/网络及 HDZC `deploy-api-1` 仍运行；日期段一致性断言通过。
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
- 本轮生产推理验收：用项目登记的专用 Ed25519 SSH 密钥只读连接；运行态 `AiPlatform__Enabled=true`、项目凭证已注入，Bridge TLS readiness 和 `GET /v1/models` 均为 HTTP 200，列表包含 `deepseek-v4-flash`。仅发送一条合成提示、关闭思考且 `max_tokens=4` 的请求，返回 HTTP 403。New API 日志给出“用户额度不足，剩余额度 ¥0.000000”；代码确认该资金预扣在渠道转发前执行。`logs` 表没有该请求号的消费记录，因此请求未到 DeepSeek、未产生上游调用或扣费。未重试、未充值，未更改生产配置、价目、渠道或服务。
- 本轮进度审查（2026-10-14）：项目仍处于 P08；当前 `PASS` 只覆盖价格编辑器修复发布。负责人确认 HDZC 商家已有余额；记录中的一条受限请求却因 New API 判定剩余额度为 ¥0.000000 返回 403，未到上游。该结果只能说明那次请求所绑定的平台账户被判余额不足，不能据此要求再次加款，也不能证明 HDZC 当前无余额；失败请求对应的平台用户/令牌映射尚未在账本中核实。负责人确认当前采用人工收款、管理员后台加余额，不启用自助充值。代码核对确认 `TopupGroupRatio` 只参与在线充值计算；后台 `add_quota` 调用 `AdjustUserQuota`，按输入额度直接调整并记录审计，不应用充值倍率。因此当前无需为手工充值读取或修改 `users.group`、`TopupGroupRatio.vip`。下一步应在获准的只读范围内核对失败请求对应的平台账户与余额，不补款、不重试。本次未连接数据库、未请求模型或修改生产。
- 最小验证（2026-10-14）：完整读取 `.agents/rules/billing.md`；核对在线充值与后台加余额代码路径及计费说明；根据负责人补充确认修正 403 的解释；未运行代码测试，未访问生产数据库。
- 本轮 AI 功能窗口验证（2026-10-14）：重试后成功读取当前 Chrome 窗口，页面为鲲碟聘管理后台；“AI 账户”和“AI 分析”入口均禁用，提示“整体部署完成后恢复”。页面当前不能调用 AI；未改设置、未发送模型请求、未产生扣费。需先核实并完成该页面提示的整体部署条件，入口恢复后再做一次受限验证。

## 后续待办

- Brooks-Lint 健康基线（2026-10-12）：按负责人要求，将该次历史记录及同批次发现的空 `npx` 文件作为独立诊断提交（`5e7ac6cc4`）；`npx` 用途仍未确认，暂保留，不据此推断运行行为。生产运维待办仍以本节清单为准。
- 延后事项：有，注册入口行为复核、HDZC 实际业务调用与共享余额验收、进一步真实余额操作前的异地备份/隔离恢复、真实支付明确暂不实施并保留为后续待办；本轮已核实一次合成请求的平台消费记录和钱包扣款，未查询 DeepSeek 官方账户实际账单。2026-10-10 只读快照中的账号组 `default` 与 `TopupGroupRatio.vip=0.67` 是历史值；负责人确认当前充值由管理员手工加余额，故当前无需为该流程重新读取或调整充值倍率。只有启用在线自助充值时才需重新核实账号组与倍率。先前本机 SSH 口令的可逆编码出现在工具错误输出，负责人暂缓轮换；本轮按其要求复用过，后续仍须轮换并更新受控本机文件。DeepSeek 节假日差异按负责人决定本轮忽略；没有机器可读官方价目源，后续价格变化通过人工对照官方页面核实。
- 待办清单：见下表，共 15 项（8 项已完成，7 项待处理）。

| ID | 事项 | 暂缓原因 | 触发条件 / 计划时间 | 优先级 | 验收结果 | 关联产物 |
| --- | --- | --- | --- | --- | --- | --- |
| TODO-20261008-AIPLATFORM-SERVER-001 | 完成生产服务器盘点及 AI 域名隔离 | 本轮已完成 | 2026-10-08 | P1 | P01/P02 通过；AI 域名独立站点生效，HDZC 入口保持正常 | `deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-REGISTRATION-001 | 复核线上关闭注册后的用户入口行为 | 线上 `/api/status` 已回读为 `false/false`；尚未验证登录页隐藏注册链接与注册 API 拒绝新注册 | 注册入口验收阶段 | P1 | 注册入口隐藏，注册 API 拒绝新注册 | `docs/生产服务器部署状态.json` |
| TODO-20261008-AIPLATFORM-CREDENTIAL-001 | 为 HDZC 创建并安全保存项目凭证 | 本轮已完成 | 2026-10-09 | P1 | 商户 Key 只存于 Bridge 受控配置；项目凭证验证、只读 `GET /v1/models` 和内网 HTTPS readiness 均通过；HDZC AI 保持关闭，未调用模型 | `docs/项目凭证管理.md`；`deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-BRIDGE-IPPORT-001 | 配置并验证 Bridge 内网域名 HTTPS 入口 | 本轮已完成 | 2026-10-08 | P1 | HDZC 容器通过 `https://<BRIDGE_INTERNAL_FQDN>:3001` 完成 TLS 1.2/1.3、证书链和主机名校验；宿主机无端口映射。正式凭证与业务计费另由 CREDENTIAL TODO 跟踪 | `deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-BRIDGE-UPSTREAM-TLS-001 | 部署并验证 Bridge 到 New API 的 HTTPS 上游链路 | 本轮已完成 | 2026-10-08 | P1 | 专用证书与续期模拟通过；内网临时探针经证书链和主机名校验，在 TLS 1.2/1.3 下到达 New API；不发布宿主机端口，AI 门户及 HDZC 现有功能正常。正式 Bridge 请求与凭证验收由 CREDENTIAL TODO 跟踪 | `deploy/production/生产对照核验.md` |
| TODO-20261009-AIPLATFORM-BRIDGE-FIREWALL-001 | 安装 Bridge 防火墙 systemd 单元并验证 Docker 重启恢复 | 本轮已完成：2026-10-10 已备份并修正生产 Bridge 监听地址，安装并启用防火墙与 Bridge 单元，Docker 重启后服务恢复 | 已完成：2026-10-10 | P1 | Compose、systemd 与 `iptables -C` 核验通过；HDZC readiness HTTP 200；Docker 重启后原运行容器全部恢复。非 HDZC 网络探测被 Docker 网络隔离提前阻断，DROP 规则计数未增加，未独立观测到该规则命中 | `deploy/production/bridge-firewall.sh`；`deploy/production/systemd/ai-platform-bridge-firewall.service`；`deploy/production/生产对照核验.md` |
| TODO-20261008-AIPLATFORM-BACKUP-001 | 增加服务器外备份并在真实业务 schema/数据上复验恢复 | 当前同机备份仅覆盖首次启动前的空库；服务器故障可能同时影响主数据和备份 | 开放真实充值前 | P1 | 服务器外备份、保留链及业务数据隔离恢复全部通过 | `docs/生产服务器部署状态.json` |
| TODO-20261008-AIPLATFORM-PAYMENT-MODEL-001 | 验收真实支付及 HDZC 共享余额业务能力 | 用户明确要求真实支付先记录为待办、当前不实施；共享余额业务调用（HDZC）也尚未完成 | 用户重新安排真实支付验收时；共享余额按单独授权和凭据就绪后验收 | P1 | 真实支付暂不验收；后续需验证支付回调、余额到账与退款边界。HDZC 共享余额业务调用及扣费审计另行验收 | `docs/部署与凭据说明.md` |
| TODO-20261009-AIPLATFORM-DEEPSEEK-001 | 仅配置 DeepSeek V4 Flash 通道与官方价目 | 2026-10-10 只读核验确认生产单价与官方当前数值一致；负责人决定本轮忽略节假日差异，不调整现有表达式。官方文档未公布机器可读价目 API，内置同步返回 401；不开发未文档化抓取、不使用渠道密钥同步，价格变化采用人工对照官方页面 | 官方价目发生变化或业务负责人要求调整计价规则时复核 | P1 | 当前仅配置 DeepSeek V4 Flash，单价匹配官方当前价目；节假日峰时可能按官方闲时价两倍计费，此差异按负责人决定暂时接受；不改其他模型或全局汇率。生产调用与计费另由本轮新待办跟踪 | `pkg/billingexpr/expr.md`；DeepSeek 官方价目页 |
| TODO-20261009-AIPLATFORM-PRICING-EDITOR-DEPLOY-001 | 发布并验证后台价格编辑器修复 | 本地修复和验证完成，Certbot 模拟及发布门禁通过；已生产部署并完成管理员浏览器验收 | 已完成：2026-10-10 | P1 | 生产“模型广场”保持关闭时，管理员仍可打开 `deepseek-v4-flash` 编辑窗口；没有 `pricing is disabled` 提示。没有保存价目或发起模型调用、扣费 | `web/src/features/model-pricing/model-pricing-panel.tsx`；`web/src/features/model-pricing/__tests__/editor-layout.test.tsx` |
| TODO-20261010-AIPLATFORM-TLS-CERT-MOUNT-001 | 将 New API TLS 证书源移出可写数据挂载并发布 | 本轮已完成：2026-10-10 已迁移目录、更新 Compose 与 Certbot 钩子、通过运行态核验及续期模拟 | 已完成：2026-10-10 | P1 | 证书源目录位于 New API 可写数据挂载之外；Certbot 写入路径与只读挂载一致；New API 健康检查为 `healthy`，TLS 1.2/1.3 和依赖服务 readiness 通过；迁移后 Certbot 3.3.0 续期模拟成功 | `deploy/production/docker-compose.yml`；`deploy/production/certbot/21-deploy-ai-platform-new-api-cert.sh`；`docs/部署与凭据说明.md`；`deploy/production/生产对照核验.md` |
| TODO-20261010-AIPLATFORM-OCR-FINAL-REVIEW-001 | 在下一独立审查周期核对 R12 修复 | 本轮已完成：独立 OCR 复核 `AGENTS.md` 与 `test_bridge_deployment_config.py`，2/2 完成、0 条评论、0 失败 | 已完成：2026-10-10 | P1 | 已核对两项修复；本结果只覆盖所选两个文件，不代表整个仓库审查通过 | docs/reports/2026-10-09-ocr-review.md |
| TODO-20261010-AIPLATFORM-DEEPSEEK-SMOKE-001 | 排查 DeepSeek V4 Flash 生产请求 403 并完成一次计费验收 | 平台侧单次成功请求、消费记录及钱包扣款已只读对账；未查询 DeepSeek 官方账户实际账单，真实 HDZC 业务场景仍待验收 | 需要核对 DeepSeek 官方账户余额或验收真实 HDZC 业务时 | P1 | 平台日志记录 8/1 tokens、1 额度点；账户 used_quota 与请求数分别为 1；未重复请求 | `.project-workflow/PROJECT.md` |
| TODO-20261010-AIPLATFORM-CNY-VIP-TOPUP-001 | 记录人民币余额口径；仅在启用自助充值时评估 VIP 充值倍率 | 当前人工收款并由管理员按约定直接加余额；后台加余额不使用 `TopupGroupRatio`，因此当前不需要读取或调整该倍率。在线支付未启用。人民币费率严格对齐官方价目尚未被要求 | 仅在启用在线自助充值，或负责人明确要求人民币费率严格匹配官方时评估 | 当前手工充值按约定目标金额换算额度点后录入；历史倍率 `0.67` 不作为当前值，亦不阻塞手工流程。若未来启用自助充值，再核实账号组与倍率 | `docs/AI平台计费配置与计算规则.md` |
| TODO-20261010-AIPLATFORM-SSH-CREDENTIAL-ROTATE-001 | 轮换生产 SSH 登录口令并更新受控本机凭据 | 工具错误输出曾包含口令的可逆编码；负责人暂缓轮换。本轮按负责人要求复用受控本机文件中的旧口令，只执行只读查询 | 负责人可操作时 | P1 | 旧口令失效；新口令仅保存于服务器及权限 `0600` 的本机受控文件，未在聊天或日志回显 | `/Users/zhoudai/Downloads/Projects/hedezhongcheng/server/production.local.json` |

## 前序发布门禁（2026-10-10）

- 结论：PASS
- 条件：仅针对后台价格编辑器修复发布。Certbot 3.3.0 续期模拟成功；前端全量测试、生产构建、候选镜像传输校验和生产 Compose 校验通过；回滚文件与旧镜像保留。生产 New API 已 healthy，内外 `/api/status` 均为 HTTP 200，其他 AI 平台服务容器未重启。
- 发布证据：生产只更改 `AI_PLATFORM_RELEASE` 并仅重建 `new-api`。候选镜像 `ai-platform-new-api:pricing-editor-20261010-cdc9e7c82-r1` 的配置摘要和 5 层 diff ID 经服务器导出回读，与本地构建一致；管理员浏览器中 `deepseek-v4-flash` 定价编辑器成功打开，未显示 `pricing is disabled`。回滚副本位于 `<AI平台备份目录>/releases/pre-pricing-editor-20261010T074837Z`，权限 `0600 root:root`；旧镜像 `ai-platform-new-api:tls-c69fcdd9cab1` 保留，可恢复旧标签并仅重建 `new-api`。
- 阻塞项：本次价格编辑器发布无剩余阻塞。单次受限合成请求经 Bridge 返回 HTTP 200，DeepSeek 返回模型标识 `deepseek-flash`，用量为输入 8、输出 1、合计 9 tokens；New API 生产日志与钱包扣款已核对。未改生产价目或设置、未重启 HDZC；DeepSeek 官方账户账单、真实 HDZC 业务调用仍未验收，真实支付明确列为后续待办且当前不实施。CNY 官方单价口径和 VIP 五折设置另有待办；节假日差异按负责人决定暂时忽略。

## 数据库证据

- db-mcp 状态：未连接；本轮未使用 `127.0.0.1:13306`。通过受控 SSH 在 AI 平台 PostgreSQL 执行只读事务查询。
- 数据范围与业务口径：只读查询该次请求对应的 `logs`、`users`、`tokens` 与计费选项；未访问 HDZC MySQL，未修改数据库结构或数据。
- 请求对账：`prod-smoke-balance-20261010-01` 唯一记录为 `deepseek-v4-flash`，时间 2026-10-10 17:06:53（北京时间），8 输入、1 输出、缓存 0、匹配 `off_peak`、扣除 1 额度点。对应账户剩余 357142 点、累计使用 1 点、请求 1 次；令牌为不限额。账户用户组为 `default`，此次请求组为 `vip`。
- 计费配置：生产 `tiered_expr` 的 USD 单价与 DeepSeek 官方当前美元价一致；`USDExchangeRate=7`、`Price=7`、`GroupRatio.vip=1`、`TopupGroupRatio.vip=0.67`，默认充值倍率未配置并按代码回退为 1。该账户属于 `default`，因此 VIP 充值倍率不会因请求令牌的 `vip` 分组而生效。CNY 显示余额为 `357142 / 500000 × 7 = ¥4.999988`，页面显示 ¥5；实际消费 1 点按汇率折算为 ¥0.000014，页面 4 位小数的非零最小显示规则将其显示为 ¥0.0001。
- 结构与数据：未执行数据库结构变更或直接写库；未调用第二次模型、未执行充值或支付。DeepSeek 官方账户账单及 HDZC 真实业务链路仍未核验；SSH 登录口令按负责人此前指示使用旧值，尚未轮换。

## 前序 Transition（2026-10-10）

- Flow：7. 运行与优化 / 生产部署准备 P08（Bridge 防火墙与 Docker 重启恢复已验证；DeepSeek 节假日差异本轮忽略）
- Gate：PASS（后台价格编辑器修复已部署并通过管理员页面验收；Certbot 迁移后续期模拟和生产健康检查通过）。
- Next：当前 Chrome 打开的鲲碟聘管理后台将“AI 账户”和“AI 分析”标为禁用，提示“整体部署完成后恢复”；先核实并完成该页面提示的整体部署条件，入口恢复后再发一条 `max_tokens=4` 合成请求，不绕过禁用状态。负责人确认 HDZC 已有余额，不再建议加款；待生产 PostgreSQL 只读访问获批后，再核对最近 403 请求绑定的平台用户、令牌所属用户及该用户余额，解释它与 HDZC 现有余额/既有成功请求的差异。当前人工加余额流程无需读取或修改 `users.group`、`TopupGroupRatio.vip`。在线自助充值和真实支付暂不实施；后续新增真实资金操作前仍须完成异地备份与隔离恢复，旧 SSH 口令待轮换。

## 历史记录

| 时间 | 阶段 | 动作 | 结果 |
| --- | --- | --- | --- |
| 2026-10-11 | 7. 运行与优化 / 生产部署准备 P08 | 整理本轮变更并原子提交、统一推送 | 将 10 个工作区变更文件按生产核验、计费规则文档、OCR 复核和工作流台账拆成 4 个独立提交；Bridge 配置回归 18/18、状态 JSON 解析和差异检查通过。所有提交完成后统一推送；未改生产配置或发起新的模型请求。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 启动 DeepSeek 限量生产请求验收并排查 HTTP 403 | 生产定价页面回读到既有峰时/闲时表达式，价格未修改；专用 Ed25519 SSH 密钥连接成功，未使用旧口令。运行态 AI 开关与项目凭证已配置；Bridge TLS readiness、带凭证 `GET /v1/models` 均 HTTP 200，模型列表包含 `deepseek-v4-flash`。一条关闭思考、`max_tokens=4` 的合成请求返回 HTTP 403。New API 日志确认商户用户余额为 ¥0.000000；源代码确认余额预扣在上游转发前完成，因此该请求未到 DeepSeek、没有上游调用或费用。未重试、未充值、未改生产配置或价目。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 余额增加后完成一次 DeepSeek 限量复测 | 负责人反馈已增加余额；未代充值。使用 HDZC 运行容器中的项目凭证经 Bridge 发送一条关闭思考、`max_tokens=4` 的合成请求，HTTP 200；返回模型 `deepseek-flash`，用量 8 输入、1 输出 token。请求号 `prod-smoke-balance-20261010-01`；未回显响应正文、未重复调用或改配置/价目。按负责人此前要求使用受控本机文件中的旧 SSH 口令，仍待轮换。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 只读对账 DeepSeek 合成请求、余额与 VIP 充值设置 | 请求唯一记录 8/1 tokens、缓存 0、off-peak、消费 1 点；账户 used_quota=1、request_count=1、余额 357142 点，与页面 ¥5 一致。页面 ¥0.0001 是四位小数非零显示下限；实际按应用汇率折算为 ¥0.000014。发现官方人民币价与应用汇率 7 的口径差异；VIP 账户组为 default、VIP 充值倍率 0.67，未达五折。只读查询，无生产写入、第二次模型调用、充值或支付。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 补充确认 ¥5 余额的入账方式并整理计费规则 | 负责人确认此前收到 ¥5 后由管理员手动增加商户余额；当前人工模式按实付金额的 2 倍加余额，绕过在线充值倍率。计费公式和配置位置已写入 `docs/AI平台计费配置与计算规则.md`；未修改生产设置、支付或余额。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 运行 Certbot 续期模拟并发布后台价格编辑器修复 | systemd 管理的 Certbot 3.3.0 dry-run 成功；发布门禁通过后只更新 `AI_PLATFORM_RELEASE` 并重建 `new-api`。New API healthy，本机与公网 API 状态均 HTTP 200；Bridge、PostgreSQL、Redis 容器未重启。管理员浏览器中 `deepseek-v4-flash` 编辑表单和预览正常；未保存价格、调用模型或扣费。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 安装 Bridge 防火墙单元并验证 Docker 重启恢复 | 生产 Compose 由全网卡监听改为 HDZC 网络固定地址，变更前文件、systemd 单元和 iptables 专用链已备份；防火墙和 Bridge 单元启用，规则挂接 `DOCKER-USER`。Docker 重启后 5 个原运行容器全部恢复，New API/PostgreSQL/Redis healthy，Bridge readiness 与公网 HDZC API readiness 均 HTTP 200。另网段 TCP 探测被 Docker 网络隔离阻断，DROP 计数未增加；运行容器 `AiPlatform__Enabled=true`。未发起推理或扣费。 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 更正 Bridge 入口记录 | 不使用 `<Bridge内网域名>`；仅将 HDZC 可达的内网 IP + `3001` 作为目标，当前宿主机尚未映射端口，未做生产网络变更 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 通过隔离探针选择并验证 Bridge 网络入口 | HDZC 容器访问宿主机 `<AI平台服务器内网地址>:3001` 超时；访问共享 Docker 网络地址 `<Bridge容器内网地址>:3001` readiness 返回 200；生产 Compose 不发布宿主机端口，正式凭证接入仍待完成 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 核对 HDZC 生产 HTTPS 门禁 | HDZC 启用 AI 时拒绝 HTTP 地址；Bridge 当前仅支持 HTTP。未改 HDZC 配置或代码、未启动正式 Bridge；后续先建立可信内网 HTTPS |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 建立 Bridge 专用证书和内网 HTTPS | 专用证书已签发；Docker 内网域名别名生效；HDZC 容器 TLS 1.2/1.3 readiness 验收通过；探针已清理。正式凭证和 HDZC 配置仍待处理 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 准备 Bridge→New API HTTPS 上游链路 | 本地 New API TLS 监听、生产 Compose 和专用续期钩子已准备；定向测试及 Compose 校验通过；SSH 公钥认证失败，因此未在服务器签发证书、发布镜像或重建服务；生产仍需部署验收 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 按 HDZC 受控连接说明恢复 SSH 并部署 Bridge→New API HTTPS | 备份 AI PostgreSQL、原 Compose 与生产环境；签发专用证书并通过续期模拟；发布 New API TLS 镜像，TLS 1.2/1.3 内网探针通过；AI 门户和 HDZC 健康回读通过，HDZC 配置/数据库未动。Bridge 仍 disabled；预先存在的企业 Token 字段未读取或修改，归属待核实 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 比对 HDZC 与 AI Bridge 本地及线上配置 | 本地 HDZC AI 保持关闭、项目 ID 为 `hedezhongcheng`、企业标识为 `platform-ops`；线上 HDZC 只有 `AiPlatform__Enabled=false`。线上 AI Bridge 项目 ID 和企业映射均为 `NOT_CONFIGURED`；Bridge disabled/inactive。AI 门户与 HDZC API 健康检查 200；未改生产配置、未访问 HDZC 数据库、未调用模型或扣费。商户 Key 在页面遮挡显示，本地旧 POC Key 与其不匹配 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P00 | 初始化项目工作流，记录共用登录网址、内部 Bridge、DNS 和预上线备份基线 | P00 完成；进入 P01 只读盘点，真实充值前仍需异地备份 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P01 | 通过 HDZC 项目受控 SSH 配置只读盘点主机资源、数据盘、Docker、Nginx、时间、监听端口和 UFW | P01 通过；未连接数据库，未改 HDZC 服务；进入 P02 HTTPS 核验 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P02 | 核验 DNS、证书与续期模拟；新增 AI 独立 Nginx 站点并 reload | AI HTTP 301、HTTPS 503 占位且 TLS 有效；HDZC 管理/API 检查通过，原站点配置哈希未变；进入 P03 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P03 | 完成前端、后端和 Bridge 构建核验；固定镜像与 Compose 网络边界；扫描发布配置和镜像归档 | P03 门禁通过，镜像已上传并在服务器加载；归档摘要未留存，当前记录无法复核传输一致性；未启动 AI 服务，进入 P04 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P04 | 生成服务器侧随机数据库和会话秘密并设置 root-only 权限；校验生产 Compose | 实际秘密值未回读；Compose 两种 profile 校验通过；HDZC 原容器仍运行，AI 服务尚未启动；进入 P05 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P05 | 启动 AI 独立 PostgreSQL/Redis；启用每日备份 timer；完成 New API schema 备份和 tmpfs 隔离恢复 | PG/Redis healthy，备份 service success/0，40 张表恢复一致，HDZC 容器仍运行；进入 P07，支付/模型后置 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P07 | 启动 New API、切换 AI 专属 Nginx 到本机反向代理并验证公网健康；保存 503 占位配置用于回滚 | AI 门户/API 200，HDZC 管理/API 正常且原站点哈希不变；首个管理员待负责人自行创建 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P07 | 负责人创建首个平台管理员；AI 站点根路径改为 302 跳转 `/sign-in`，避免访客看到 New API 介绍页 | root 管理员计数为 1；公网 `/` 为 302 至 `https://<AI平台公网域名>/sign-in`，登录页和 `/api/status` 为 200；`nginx -t` 与 reload 通过；回滚配置保存在 `<AI平台备份目录>/nginx/ai-platform-before-login-redirect-20261008T082408Z.conf`；本轮未修改 HDZC Nginx 配置，API 状态正常；进入 P08 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 只读比较本地与线上 New API 有效注册选项及 AI Nginx 文件 | 本地注册/密码注册为 false/false，线上为 true/true；AI Nginx 配置摘要未留存，当前记录无法复核内容一致性；公网根路径跳转一致；未写线上设置，待管理会话和写入授权后同步注册开关 |
| 2026-10-08 | 7. 运行与优化 / 生产部署准备 P08 | 回读公网注册开关并核对商户接入信息；负责人更正 Bridge 入口为服务器内网 IP + 端口 | 线上注册/密码注册为 false/false；负责人确认商户账号分组为 `vip`，截图确认商户 API Key 启用且令牌分组为 `vip`；密钥未读取或传递；Bridge 容器监听 `3001`，生产宿主机端口映射、readiness、充值和模型调用未验证；先前域名解析检查不再作为验收条件 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 按用户请求重载 CUA Driver MCP | 配置与 `codex mcp list` 显示 `cua-driver` 已启用且含 `--grant existing-profile`；用户已报告重启，但当前任务和原任务的工具目录仍未加载该 MCP。没有读取 Chrome 页面或密钥、没有改线上配置。下一步需在重启后的 Codex 中新建或打开任务，让工具目录重新加载后继续核验 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 管理员会话就绪后只读核验 HDZC 项目凭证 | 管理员控制台及受保护的凭证列表接口已核实；接口返回 HTTP 200，`hedezhongcheng` 无记录。未显示或签发密钥、未改生产配置；本轮服务器侧 AI 开关未重新回读，最近有效证据仍是 2026-10-08 的 `false`。Bridge 配置需等项目凭证操作确认及商户 Key 核验 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 配置并启用 Bridge 商户映射 | Secret 文件脱敏核验与 Compose 校验通过；Bridge systemd enabled/active，内网 HTTPS readiness HTTP 200；只读 GET /v1/models HTTP 200。HDZC AI 仍关闭，未重启 HDZC、未调用模型推理或扣费。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 生产 Bridge 命名统一为 `hdzc` 并保留 `hzc` 兼容 | 首轮检查失败后自动恢复原配置；第二轮备份后将 systemd 标准 profile 切换为 `hdzc`，Compose 同时保留 `hzc`。两个 profile 均验证通过，Bridge active/readiness HTTP 200；HDZC AI 仍关闭且 `deploy-api-1` 未重启。备份：`<AI平台备份目录>/hdzc-profile-20261009T045804Z-870254`。 |
| 2026-10-09 | 7. 运行与优化 / 生产部署准备 P08 | 仅接入 DeepSeek V4 Flash 并配置官方现行价目 | 负责人报告已输入官方 API Key；本轮 CUA 连续超时，未能确认保存或继续设置价格，未读取或提交密钥，也未改生产渠道/价目。当前阻塞及待办已记录，HDZC AI 开关只沿用上一轮回读、未重新验证。 |
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
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 独立复核 R12 两项修复 | OCR 会话 `76d043be-9749-4756-ae5b-74fbe75e4a1b` 仅审查 `AGENTS.md` 与 Bridge 部署配置测试，2/2 完成、0 条评论、0 失败；DeepSeek 官方 API 直连。生产 TLS 迁移、防火墙持久化及业务验收仍未完成，门禁保持 NEED_VERIFY。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 完成主工作区原子提交 | 按模型定价编辑器、内部 HTTPS、生产部署配置、项目工作流、OCR 记录拆分为 5 个提交；未推送、未部署。工作区干净，生产门禁仍为 NEED_VERIFY。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 迁移 New API TLS 证书源目录并验收 | 证书源移出 New API 可写数据挂载；Compose 只读挂载与 Certbot 钩子路径同步，权限和运行状态核验通过。仅重建 New API 并沿用原镜像；TLS 1.2/1.3、AI 门户与 HDZC readiness 均为 200，其他服务未重启。保留仅 root 可访问的回滚备份；迁移后未执行 Certbot 续期模拟。整体发布门禁仍为 NEED_VERIFY |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 暂不处理 DeepSeek 节假日计价差异 | 负责人决定本轮忽略节假日；不修改现有计价表达式或生产配置。已知节假日峰时可能按官方闲时价两倍计费，按本次范围暂时接受；官方价目变化采用人工对照页面核实。 |
| 2026-10-10 | 7. 运行与优化 / 生产部署准备 P08 | 核对计费小数舍入原则 | 不建议对每笔账单的任意小数一律向上取整，以免按请求数系统性多收；当前表达式路径按四舍五入规则将本次 0.9 额度点记为 1 点，部分旧浮点路径会截断，页面 ¥0.0001 是非零最小显示值而非实际扣款。未改代码、生产设置或数据；用户尚未决定是否统一策略。 |
| 2026-10-14 | 7. 运行与优化 / 生产部署准备 P08 | 核对 P08 当前进度并确定最小下一步 | 当前 PASS 仅针对价格编辑器发布；最新受限推理请求在平台预扣阶段因额度不足被拒绝，未到上游。商家账号组和 VIP 充值倍率的当前值尚未获生产 PostgreSQL 只读审批回读。下一步是获批后只读核实两项；未修改代码、生产配置或数据。 |
| 2026-10-14 | 7. 运行与优化 / 生产部署准备 P08 | 根据负责人说明确认当前采用人工后台加余额 | `TopupGroupRatio` 只用于在线充值计算；管理员 `add_quota` 路径按输入额度直接调整并留审计记录。当前无需核验或修改 VIP 充值倍率；继续真实业务验收前仍须满足异地备份要求，并取得受限调用授权。未改代码、生产配置或数据库。 |
| 2026-10-14 | 7. 运行与优化 / 生产部署准备 P08 | 根据负责人确认 HDZC 已有余额，修正 403 的下一步 | 不再建议加款。最近 403 仅说明该次请求绑定的平台账户被判余额不足；需经获批的只读查询核对其用户/令牌映射和余额，并与 HDZC 已有余额及既有成功请求对账。未重试、未连接生产数据库、未改生产数据。 |
| 2026-10-14 | 7. 运行与优化 / 生产部署准备 P08 | 使用当前窗口验证 AI 功能 | 成功读取当前 Chrome 的鲲碟聘管理后台；“AI 账户”和“AI 分析”均禁用，提示整体部署完成后恢复。未改设置、未发起请求或产生扣费；待整体部署条件完成后验证。 |
