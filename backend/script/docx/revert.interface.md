你需要在网络设备（H3C / Comware 风格 CLI）上管理 NAT Server 端口映射，规则如下：

==============================

【方法一：新增 NAT Server】

==============================

操作流程：

1. 进入 system-view

2. 进入对应接口视图（如 GigabitEthernet0/0）

3. 使用完整的 nat server protocol 命令

4. 命令执行后立即生效，不需要执行 save

标准命令格式：

nat server protocol <tcp|udp> global <公网IP> <公网端口> inside <内网IP> <内网端口> description <description>

说明：

- description 建议仅使用英文、数字和短横线

- description 用于标识：服务器 / 项目 / 用途 / 版本或有效期等信息

示例：

nat server protocol tcp global 117.149.14.2 65532 inside 192.168.1.218 48080 description youxihu-test-dw-backend-port

==============================

【方法二：删除 NAT Server】

==============================

操作原则：

1. 删除必须使用 undo nat server 命令

2. undo 命令必须在接口视图下执行

3. undo 命令参数必须与原 NAT 规则严格匹配

4. 参数不完整或不匹配将导致删除失败

标准命令格式（推荐自动化使用完整匹配）：

undo nat server protocol <tcp|udp> global <公网IP> <公网端口>

示例：

undo nat server protocol tcp global 117.149.14.2 65532

==============================

【重要原则】

==============================

1. 新增和删除操作必须在接口视图（如 GigabitEthernet0/0）下执行

2. 自动化系统应先执行：

   - screen-length disable

   - display nat server

3. 解析 display nat server 输出，构建 NAT Server 结构体

4. 反向生成 undo nat server 命令执行清理

5. 上述两种方法的终极目标是：

   👉 通过删除旧端口映射 + 新增新端口映射，实现 **端口切换**