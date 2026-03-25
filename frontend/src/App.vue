<template>
  <div class="app-container">
    <div class="header">
      <h1 class="title">星目路由器端口快切工具</h1>
    </div>

    <div class="main-layout">
      <!-- 主内容区 -->
      <div class="main-content">
        <div class="console-layout">
      <!-- 状态区 -->
      <div class="status-section">
        <div class="section-header">
          <h2 class="section-title">当前端口映射状态</h2>
        </div>

        <div class="status-grid">
          <div
            v-for="port in portConfigs"
            :key="`status-${port.external_port}`"
            class="status-item"
            :class="{
              'current': portMappings[`port_${port.external_port}`] === originalMappings[`port_${port.external_port}`],
              'changed': portMappings[`port_${port.external_port}`] !== originalMappings[`port_${port.external_port}`]
            }"
          >
            <div class="status-port">{{ port.external_port }}</div>
            <el-icon class="status-arrow"><ArrowRight /></el-icon>
            <div class="status-target">{{ portMappings[`port_${port.external_port}`] || '未配置' }}</div>
            <div class="status-indicator-wrapper">
              <el-icon v-if="portMappings[`port_${port.external_port}`] === originalMappings[`port_${port.external_port}`]" class="status-ok">
                <CircleCheckFilled />
              </el-icon>
              <el-icon v-else-if="portMappings[`port_${port.external_port}`] !== originalMappings[`port_${port.external_port}`]" class="status-pending">
                <Clock />
              </el-icon>
            </div>
          </div>
        </div>
      </div>

      <!-- 配置区 -->
      <div class="config-section">
        <div class="section-header">
          <h2 class="section-title">端口映射配置</h2>
          <div class="config-status">
            <el-icon class="status-indicator" :class="{ 'synced': !hasChanges, 'pending': hasChanges }">
              <CircleCheckFilled v-if="!hasChanges" />
              <Clock v-else />
            </el-icon>
            <span class="status-text">{{ hasChanges ? '有待保存的变更' : '配置已同步' }}</span>
          </div>
        </div>

        <div class="port-cards">
          <el-card
            v-for="port in portConfigs"
            :key="port.internal_port"
            class="port-card"
            shadow="never"
          >
            <template #header>
              <div class="card-header">
                <div class="header-content">
                  <span class="port-desc">{{ port.description }}</span>
                  <span class="port-title">{{ port.external_ip }}:{{ port.external_port }}</span>
                </div>
                <el-icon class="expand-icon"><ArrowDown /></el-icon>
              </div>
            </template>

            <div class="port-options">
              <el-radio-group v-model="portMappings[`port_${port.external_port}`]" class="radio-group">
                <div
                  v-for="option in port.options"
                  :key="option.ip"
                  class="radio-item"
                >
                  <el-radio :value="option.ip" class="custom-radio">
                    <span class="ip-text">{{ option.ip }}</span>
                    <el-tag
                      class="env-tag"
                      :class="`${option.environment}-tag`"
                      size="small"
                    >
                      {{ option.environment }}
                    </el-tag>
                  </el-radio>
                </div>
              </el-radio-group>
            </div>
          </el-card>
        </div>

        <!-- 操作按钮区 - 合并到配置区内 -->
        <div class="action-area">
          <el-button
            type="primary"
            size="large"
            class="save-button"
            :class="{ 'has-changes': hasChanges, 'no-changes': !hasChanges }"
            @click="saveConfiguration"
            :loading="saving"
            :disabled="!hasChanges"
          >
            <el-icon>
              <DocumentAdd v-if="hasChanges" />
              <CircleCheckFilled v-else />
            </el-icon>
            {{ hasChanges ? '保存并应用配置' : '配置已同步' }}
          </el-button>
        </div>
      </div>
        </div>
      </div>

      <!-- 右侧悬浮日志面板 -->
      <div class="logs-sidebar">
        <OperationLogs ref="operationLogsRef" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowRight, CircleCheckFilled, DocumentAdd, Clock } from '@element-plus/icons-vue'
import axios from 'axios'
import OperationLogs from './components/OperationLogs.vue'

// API基础URL - 使用相对路径，通过nginx代理转发
const getApiBaseUrl = () => {
  // 在生产环境中，nginx会将/api路径代理到后端
  // 开发环境可以通过环境变量或配置文件指定
  if (import.meta.env.DEV) {
    // 开发环境使用动态端口
    const hostname = window.location.hostname
    const port = '8080'
    return `http://${hostname}:${port}`
  } else {
    // 生产环境使用相对路径，通过nginx代理
    return ''
  }
}

// 获取客户端标识（使用内网IP+浏览器指纹）
const getClientExternalIP = async () => {
  console.log('使用内网IP+浏览器指纹作为客户端标识')

  // 使用内网IP + 浏览器指纹作为唯一标识
  const internalIP = getClientInternalIP()

  // 生成浏览器指纹（基于用户代理、屏幕分辨率等）
  const fingerprint = btoa(
    navigator.userAgent +
    screen.width + 'x' + screen.height +
    new Date().getTimezoneOffset()
  ).substr(0, 8)

  const clientId = `${internalIP}-${fingerprint}`
  console.log('生成的客户端标识:', clientId)

  return clientId
}

// 获取客户端内网IP
const getClientInternalIP = () => {
  // 在浏览器中，我们只能获取到访问的hostname
  // 如果是内网访问，hostname就是内网IP
  const hostname = window.location.hostname
  // 检查是否是内网IP格式
  if (hostname.match(/^192\.168\.|^10\.|^172\.(1[6-9]|2[0-9]|3[0-1])\./)) {
    return hostname
  }
  return 'unknown'
}

// 初始化IP信息并配置axios默认headers
const initializeIPHeaders = async () => {
  const clientInternalIP = getClientInternalIP()
  const clientExternalIP = await getClientExternalIP()

  axios.defaults.headers.common['X-Client-Internal-IP'] = clientInternalIP
  axios.defaults.headers.common['X-Client-External-IP'] = clientExternalIP

  console.log('客户端IP信息:', {
    internal: clientInternalIP,
    external: clientExternalIP
  })
}

const API_BASE_URL = getApiBaseUrl()

// 端口配置信息
const portConfigs = ref([])

// 端口映射配置 - 使用ref而不是reactive
const portMappings = ref({})

// 原始映射状态（用于比较是否有变更）
const originalMappings = ref({})

const saving = ref(false)
const loading = ref(false)
const hasChanges = ref(false)

// 操作日志组件引用
const operationLogsRef = ref(null)

// 监听配置变更
watch(portMappings, () => {
  hasChanges.value = false
  for (const key in portMappings.value) {
    if (portMappings.value[key] !== originalMappings.value[key]) {
      hasChanges.value = true
      break
    }
  }
}, { deep: true })

// 获取端口配置信息
const fetchPortConfig = async () => {
  try {
    const response = await axios.get(`${API_BASE_URL}/api/port-config`)
    portConfigs.value = response.data.ports

    // 初始化映射状态 - 使用外网端口作为键名
    const newPortMappings = {}
    const newOriginalMappings = {}

    portConfigs.value.forEach(port => {
      const key = `port_${port.external_port}` // 使用带下划线的格式，与后端一致
      newPortMappings[key] = ''
      newOriginalMappings[key] = ''
    })

    portMappings.value = newPortMappings
    originalMappings.value = newOriginalMappings

    console.log('获取到的端口配置:', response.data)
    console.log('初始化的portMappings键:', Object.keys(portMappings.value))
  } catch (error) {
    console.error('获取端口配置失败:', error)
    ElMessage.error('获取端口配置失败，请检查后端服务')
  }
}

// 获取当前端口状态并自动填充选择
// 获取当前端口状态 - 只从Redis缓存获取
const fetchPortStatus = async () => {
  loading.value = true
  try {
    const response = await axios.get(`${API_BASE_URL}/api/port-status`)
    const status = response.data

    console.log('从缓存获取的状态:', status)
    console.log('当前portMappings键名:', Object.keys(portMappings.value))

    // 创建新的映射对象
    const newPortMappings = { ...portMappings.value }
    const newOriginalMappings = { ...originalMappings.value }

    // 更新当前映射状态
    for (const key in status) {
      // 直接使用后端返回的键名，不需要转换
      console.log(`检查键名 ${key}，值: ${status[key]}`)
      if (newPortMappings.hasOwnProperty(key)) {
        newPortMappings[key] = status[key] || ''
        newOriginalMappings[key] = status[key] || ''
        console.log(`✓ 成功更新 ${key} = ${status[key]}`)
      } else {
        console.log(`✗ 键名 ${key} 不存在于 portMappings 中`)
        console.log('可用的键名:', Object.keys(newPortMappings))
      }
    }

    console.log('更新前的portMappings:', portMappings.value)
    console.log('准备设置的newPortMappings:', newPortMappings)

    // 一次性更新所有数据
    portMappings.value = newPortMappings
    originalMappings.value = newOriginalMappings

    console.log('更新后的portMappings:', portMappings.value)

    // 等待下一个tick确保DOM更新
    await nextTick()
    console.log('DOM更新完成，最终选中状态:', portMappings.value)

  } catch (error) {
    console.error('获取端口状态失败:', error)
    ElMessage.error('获取端口状态失败，请检查后端服务')
  } finally {
    loading.value = false
  }
}

// 保存配置
const saveConfiguration = async () => {
  // 检查是否所有端口都已配置
  const allConfigured = portConfigs.value.every(port => {
    const key = `port_${port.external_port}` // 使用带下划线的格式
    return portMappings.value[key] // 使用.value访问ref
  })

  if (!allConfigured) {
    ElMessage.warning('请为所有端口选择映射地址')
    return
  }

  // 检查是否有变更
  if (!hasChanges.value) {
    ElMessage.info('没有配置变更需要保存')
    return
  }

  saving.value = true

  try {
    // 构建批量配置请求
    const configs = []

    portConfigs.value.forEach(port => {
      const key = `port_${port.external_port}` // 使用带下划线的格式，与后端一致
      if (portMappings.value[key] !== originalMappings.value[key]) {
        configs.push({
          internal_port: port.internal_port,
          internal_ip: portMappings.value[key]
        })
      }
    })

    console.log('提交的配置变更:', configs)

    // 调用批量配置API
    const response = await axios.post(`${API_BASE_URL}/api/apply-config`, {
      configs: configs
    })

    console.log('配置应用结果:', response.data)

    // 重新获取状态
    await fetchPortStatus()

    // 刷新操作日志
    if (operationLogsRef.value) {
      operationLogsRef.value.refreshLogs()
    }

    ElMessage.success('配置保存成功！')
  } catch (error) {
    console.error('保存配置失败:', error)
    if (error.response?.data?.message) {
      ElMessage.error(`保存失败: ${error.response.data.message}`)
    } else {
      ElMessage.error('保存配置失败，请重试')
    }
  } finally {
    saving.value = false
  }
}

// 页面加载时获取配置和状态
onMounted(async () => {
  await fetchPortConfig()
  await fetchPortStatus()
})
</script>
<style scoped>
/* 确保页面可以滚动，移除overflow hidden */
:global(html, body) {
  height: 100%;
  margin: 0;
  padding: 0;
}

.app-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 2vh 16px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.header {
  text-align: center;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.title {
  color: white;
  font-size: 28px;
  font-weight: 600;
  margin: 0;
  text-shadow: 0 2px 4px rgba(0,0,0,0.3);
}

.main-layout {
  display: flex;
  gap: 16px;
  max-width: 1400px;
  margin: 0 auto;
  flex: 1;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.console-layout {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  overflow: hidden;
}

/* 右侧悬浮日志面板 */
.logs-sidebar {
  width: auto;
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.2);
  overflow: hidden;
  transition: all 0.3s ease;
  height: fit-content;
  align-self: flex-start;
}

/* 配置区 */
.config-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  padding: 12px;
  flex-shrink: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 2px solid #f0f0f0;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
}

.config-status {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-indicator {
  font-size: 14px;
}

.status-indicator.synced {
  color: #52c41a;
}

.status-indicator.pending {
  color: #faad14;
}

.status-text {
  font-size: 12px;
  font-weight: 500;
  color: #666;
}

.port-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.port-card {
  border: 2px solid #e8e8e8;
  border-radius: 12px;
  transition: all 0.3s ease;
  cursor: pointer;
}

.port-card:hover {
  border-color: #409eff;
  box-shadow: 0 8px 24px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0;
}

.header-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.port-desc {
  font-size: 16px;
  color: #2c3e50;
  font-weight: 600;
  text-align: center;
}

.port-title {
  font-size: 13px;
  font-weight: 400;
  color: #999;
  text-align: center;
  font-family: 'Monaco', 'Menlo', monospace;
}

.expand-icon {
  color: #999;
  font-size: 12px;
}

.port-options {
  padding: 12px 0;
}

.radio-group {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 8px;
  transition: all 0.2s ease;
  border: 2px solid transparent;
}

.radio-item:hover {
  background-color: #f0f9ff;
  border-color: #e1f5fe;
}

.custom-radio {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 0;
}

/* 选中状态高亮 */
.custom-radio.is-checked .radio-item {
  background-color: #e6f4ff;
  border-color: #409eff;
}

.custom-radio.is-checked .ip-text {
  color: #409eff;
  font-weight: 700;
}

.ip-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 18px;
  font-weight: 600;
  color: #2c3e50;
}

.env-tag {
  font-weight: 500;
  border: none;
  font-size: 12px;
}

.dev-tag {
  background-color: #e8f5e8;
  color: #52c41a;
}

.zc-test-tag {
  background-color: #e6f4ff;
  color: #1890ff;
}

.dw-test-tag {
  background-color: #fff2e8;
  color: #fa8c16;
}

/* 状态区 */
.status-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  padding: 16px;
  text-align: center;
  flex-shrink: 0;
}

.status-grid {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  max-width: 100%;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  background: #f8f9fa;
  border: 2px solid transparent;
  transition: all 0.3s ease;
  justify-content: center;
  flex: 1;
  min-width: 0;
}

.status-item.current {
  border-color: #52c41a;
  background-color: #f0f9ff;
  box-shadow: 0 4px 12px rgba(82, 196, 26, 0.15);
}

.status-item.changed {
  border-color: #faad14;
  background-color: #fff7e6;
  box-shadow: 0 4px 12px rgba(250, 173, 20, 0.15);
}

.status-port {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 700;
  color: #2c3e50;
  font-size: 14px;
  white-space: nowrap;
}

.status-arrow {
  color: #666;
  font-size: 16px;
}

.status-target {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
  color: #333;
  font-size: 14px;
}

.status-indicator-wrapper {
  margin-left: auto;
}

.status-ok {
  color: #52c41a;
  font-size: 14px;
}

.status-pending {
  color: #faad14;
  font-size: 14px;
}

/* 操作区 - 现在在配置区内部 */
.action-area {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.save-button {
  padding: 16px 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  border: none;
  transition: all 0.3s ease;
  min-width: 200px;
}

.save-button.has-changes {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
}

.save-button.has-changes:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 30px rgba(59, 130, 246, 0.5);
}

.save-button.no-changes {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4);
}

.save-button.no-changes:hover {
  box-shadow: 0 8px 25px rgba(16, 185, 129, 0.5);
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .main-layout {
    flex-direction: column;
  }

  .logs-sidebar {
    width: 100%;
    position: relative;
    max-height: 300px;
  }

  .port-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .status-grid {
    flex-direction: column;
    align-items: center;
  }
}

@media (max-width: 768px) {
  .app-container {
    padding: 12px;
    height: 100vh;
  }

  .title {
    font-size: 24px;
  }

  .console-layout {
    gap: 12px;
  }

  .config-section,
  .status-section,
  .operation-section {
    padding: 16px;
  }

  .port-cards {
    grid-template-columns: 1fr;
  }

  .status-grid {
    flex-direction: column;
  }

  .save-button {
    width: 100%;
    padding: 12px 24px;
  }

  .ip-text {
    font-size: 14px;
  }

  .status-port {
    font-size: 16px;
  }
}
</style>