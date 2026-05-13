# 星目端口快切工具 - 前端

基于 Vue 3 + Element Plus + Vite 的路由器端口映射管理界面。

## 目录结构

```
frontend/src/
├── api/
│   └── index.js                 # API层：axios实例、客户端IP识别、所有HTTP请求函数
├── composables/
│   ├── useTheme.js              # 主题切换（亮色/暗色，localStorage持久化）
│   └── usePortConfig.js         # 端口配置核心逻辑（fetch、save、变更检测、展开折叠）
├── components/
│   └── OperationLogs.vue        # 操作日志面板组件
├── router/
│   └── index.js                 # vue-router 路由配置
├── views/
│   └── PortManagement.vue       # 端口管理页面（主界面）
├── App.vue                      # 根组件（仅挂载 router-view）
└── main.js                      # 入口（Vue实例、ElementPlus、Router注册）
```

## 分层说明

### api 层
封装所有与后端的 HTTP 通信，包括：
- `getApiBaseUrl()` — 根据环境自动计算后端地址（开发环境动态拼接，生产环境使用相对路径）
- `getClientInternalIP()` / `getClientExternalIP()` — 客户端IP识别（内网IP+浏览器指纹）
- `initializeIPHeaders()` — 初始化 axios 默认 headers，注入客户端标识
- `fetchPortConfig()` / `fetchPortStatus()` / `applyConfig()` — 端口配置、状态查询、批量应用

### composables 层
Vue 3 Composition API 的逻辑复用单元，每个 composable 封装一类业务逻辑：
- `useTheme` — 主题状态（ref）+ 切换函数 + provide/inject 传递
- `usePortConfig` — 端口配置全流程：加载配置、加载状态、变更检测（watch deep）、保存配置、展开折叠管理

### views 层
页面级组件，对应路由的每个页面：
- `PortManagement.vue` — 当前唯一页面，包含端口状态展示、配置选择、保存按钮、操作日志

### components 层
可复用的 UI 组件，从页面中拆分出的独立功能块：
- `OperationLogs.vue` — 操作日志查询、展示、分页

### router 层
路由配置，新增页面只需：
1. 在 `views/` 下创建 Vue 文件
2. 在 `router/index.js` 添加路由条目
3. `<router-view />` 自动渲染

## 扩展指南

### 新增页面
```js
// router/index.js
const routes = [
  { path: '/', name: 'PortManagement', component: PortManagement },
  { path: '/new-page', name: 'NewPage', component: () => import('../views/NewPage.vue') }
]
```

### 新增 composable
在 `composables/` 下新建文件，遵循 `useXxx` 命名规范，返回 ref/reactive 和操作函数。

### 新增 API 函数
在 `api/index.js` 中添加函数，所有请求共用同一个 axios 实例和 API_BASE_URL 计算。

## 开发

```bash
npm install
npm run dev          # 启动开发服务器（默认 localhost:3000）
npm run build        # 构建生产版本
```

## 与后端交互

前端通过 axios 调用后端 REST API，开发环境自动拼接 `http://{hostname}:8080`，生产环境依赖 nginx 代理 `/api` 路径到后端服务。

每次请求自动携带 `X-Client-Internal-IP` 和 `X-Client-External-IP` headers，用于后端操作日志记录操作者来源。