<template>
  <div class="app-container" :class="{ 'dark-theme': isDarkTheme }">
    <div class="header">
      <h1 class="title">星目路由器端口快切工具</h1>
      <div class="header-actions">
        <div class="theme-toggle" @click="toggleTheme" :title="isDarkTheme ? '切换为亮色模式' : '切换为暗色模式'">
          <el-icon :size="18"><Moon v-if="isDarkTheme" /><Sunny v-else /></el-icon>
        </div>
        <a class="doc-link" :href="docUrl" target="_blank" title="查看说明文档">
          <el-icon :size="18"><Document /></el-icon>
        </a>
      </div>
    </div>

    <div class="main-layout">
      <div class="main-content">
        <div class="console-layout">
          <!-- 顶部状态区：只读全局概览 -->
          <div class="status-section">
            <div class="section-header">
              <h2 class="section-title">当前端口映射状态</h2>
            </div>

            <div class="status-grid-wrap">
              <div
                v-for="port in allPorts"
                :key="`status-${port.external_port}`"
                class="status-item"
                :class="{
                  'configured': getRealTargetIp(port.external_port),
                  'not-configured': !getRealTargetIp(port.external_port)
                }"
              >
                <div class="status-port">{{ port.external_port }}</div>
                <el-icon class="status-arrow"><ArrowRight /></el-icon>
                <div class="status-target">
                  {{ getRealTargetIp(port.external_port) || '未配置' }}
                </div>
                <div class="status-indicator-wrapper">
                  <el-icon v-if="getRealTargetIp(port.external_port)" class="status-ok">
                    <CircleCheckFilled />
                  </el-icon>
                  <el-icon v-else class="status-empty">
                    <Minus />
                  </el-icon>
                </div>
              </div>
            </div>
          </div>

          <div class="main-workspace">
            <!-- 左侧：端口选择列表 -->
            <div class="left-list-section">
              <div class="section-header">
                <h2 class="section-title">端口映射列表</h2>
                <span class="section-tip">再次点击可取消选择</span>
              </div>
              <div v-if="hasChanges" class="batch-action-section">
                <el-button type="primary" :loading="isSwitching || saving" :disabled="isSwitching || saving" @click="confirmSave">
                  批量保存 ({{ getPendingCount() }} 项)
                </el-button>
              </div>
              <div class="port-list">
                <div
                  v-for="port in allPorts"
                  :key="port.external_port"
                  class="port-list-item"
                  :class="{
                    'selected': selectedPortId === port.external_port,
                    'has-pending': getPortPendingTarget(port.external_port)
                  }"
                  @click="selectPort(port.external_port)"
                >
                  <div class="item-main">
                    <span class="port-desc">{{ port.displayDescription }}</span>
                    <span class="port-address">{{ port.external_ip }}:{{ port.external_port }}</span>
                    <span v-if="getPortPendingTarget(port.external_port)" class="port-pending">
                      → {{ getPortPendingTarget(port.external_port) }}
                    </span>
                  </div>
                  <div class="item-status">
                    <el-tag v-if="getRealTargetIp(port.external_port)" type="success" size="small">已配置</el-tag>
                    <el-tag v-else type="info" size="small">未配置</el-tag>
                  </div>
                </div>
              </div>
            </div>

            <!-- 右侧：详情与操作区 -->
            <div class="right-detail-section">
              <div v-if="selectedPort" class="detail-content">
                <div class="detail-header">
                  <h3 class="detail-title">{{ selectedPort.displayDescription }}</h3>
                  <span class="detail-address">{{ selectedPort.external_ip }}:{{ selectedPort.external_port }}</span>
                </div>

                <!-- 当前生效状态：只读 -->
                <div class="current-status-section">
                  <div class="section-title-small">当前生效状态</div>
                  <div class="current-status-card">
                    <div class="status-info">
                      <span class="status-ip">
                        {{ getRealTargetIp(selectedPort.external_port) || '未配置' }}
                        <span v-if="getRealTargetIp(selectedPort.external_port)">:{{ selectedPort.internal_port }}</span>
                      </span>
                      <el-tag 
                        v-if="getRealTargetEnv(selectedPort.external_port)" 
                        class="status-env" 
                        :class="`${getRealTargetEnv(selectedPort.external_port)}-tag`" 
                        size="small"
                      >
                        {{ getRealTargetEnv(selectedPort.external_port) }}
                      </el-tag>
                    </div>
                    <el-tag type="success" size="small">当前生效</el-tag>
                  </div>
                </div>

                <!-- 暂存目标（如果有） -->
                <div v-if="selectedPortTarget" class="pending-status-section">
                  <div class="section-title-small">暂存目标</div>
                  <div class="pending-status-card">
                    <div class="status-info">
                      <span class="status-ip">
                        {{ selectedPortTarget }}:{{ selectedPort.internal_port }}
                      </span>
                      <el-tag 
                        v-if="getOptionEnv(selectedPort.external_port, selectedPortTarget)" 
                        class="status-env" 
                        :class="`${getOptionEnv(selectedPort.external_port, selectedPortTarget)}-tag`" 
                        size="small"
                      >
                        {{ getOptionEnv(selectedPort.external_port, selectedPortTarget) }}
                      </el-tag>
                    </div>
                    <el-button type="danger" size="small" link @click="clearPortTarget" :disabled="isSwitching">
                      清除
                    </el-button>
                  </div>
                </div>

                <!-- 可切换目标：选择候选 -->
                <div v-if="hasCandidates(selectedPort.external_port)" class="candidates-section">
                  <div class="section-title-small">可切换目标</div>
                  <div class="candidates-list">
                    <div
                      v-for="option in getCandidates(selectedPort.external_port)"
                      :key="option.ip"
                      class="candidate-item"
                      :class="{ 'selected': selectedPortTarget === option.ip }"
                      @click="!isSwitching && selectTarget(option.ip)"
                    >
                      <div class="candidate-content">
                        <span class="candidate-ip">{{ option.ip }}:{{ selectedPort.internal_port }}</span>
                        <el-tag class="candidate-env" :class="`${option.environment}-tag`" size="small">
                          {{ option.environment }}
                        </el-tag>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-else class="no-candidates-section">
                  <el-empty description="无可切换目标" :image-size="60" />
                </div>
              </div>

              <div v-else class="empty-detail">
                <el-empty description="请先选择一个端口" :image-size="60" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="logs-sidebar">
        <OperationLogs ref="operationLogsRef" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onMounted } from 'vue'
import { ArrowRight, CircleCheckFilled, Minus, Sunny, Moon, Document } from '@element-plus/icons-vue'
import { useTheme } from '../composables/useTheme'
import { usePortConfig } from '../composables/usePortConfig'
import { initializeIPHeaders } from '../api'
import OperationLogs from '../components/OperationLogs.vue'

const { isDarkTheme, toggleTheme } = useTheme()
const {
  portConfigs,
  portMappings,
  originalMappings,
  saving,
  loading,
  hasChanges,
  operationLogsRef,
  loadPortConfig,
  loadPortStatus,
  switchPort,
  saveConfiguration
} = usePortConfig()

const docUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/readme.html`

const selectedPortId = ref(null)
const isSwitching = ref(false)

const allPorts = computed(() => {
  const ports = []
  if (portConfigs.value && Array.isArray(portConfigs.value)) {
    portConfigs.value.forEach(p => {
      if (p.all_external_ports && p.all_external_ports.length > 0) {
        p.all_external_ports.forEach((extPort, index) => {
          const displayDescription = p.all_external_ports.length > 1 
            ? `${p.description} ${index + 1}`
            : p.description
          ports.push({
            ...p,
            external_port: extPort,
            displayDescription: displayDescription
          })
        })
      } else {
        ports.push({
          ...p,
          displayDescription: p.description
        })
      }
    })
  }
  return ports
})

const selectedPort = computed(() => {
  return allPorts.value.find(p => p.external_port === selectedPortId.value)
})

const selectedPortTarget = computed(() => {
  if (!selectedPortId.value) return null
  return portMappings.value[`port_${selectedPortId.value}`] || null
})

function selectPort(externalPort) {
  if (isSwitching.value) return
  // 如果点击已经选中的端口，则取消选择
  if (selectedPortId.value === externalPort) {
    selectedPortId.value = null
  } else {
    selectedPortId.value = externalPort
  }
}

function selectTarget(ip) {
  if (!selectedPortId.value) return
  portMappings.value[`port_${selectedPortId.value}`] = ip
}

function clearPortTarget() {
  if (!selectedPortId.value) return
  portMappings.value[`port_${selectedPortId.value}`] = ''
}

function getPendingCount() {
  let count = 0
  for (const key in portMappings.value) {
    if (portMappings.value[key] && portMappings.value[key] !== originalMappings.value[key]) {
      count++
    }
  }
  return count
}

function getPortPendingTarget(externalPort) {
  const target = portMappings.value[`port_${externalPort}`]
  if (!target || target === originalMappings.value[`port_${externalPort}`]) {
    return null
  }
  return target
}

// 只获取真实的当前生效状态（来自 originalMappings）
function getRealTargetIp(externalPort) {
  if (!originalMappings.value) return ''
  return originalMappings.value[`port_${externalPort}`] || ''
}

function getRealTargetEnv(externalPort) {
  const ip = getRealTargetIp(externalPort)
  const port = allPorts.value.find(p => p.external_port === externalPort)
  if (!port || !port.options) return ''
  const option = port.options.find(o => o.ip === ip)
  return option ? option.environment : ''
}

function getOptionEnv(externalPort, ip) {
  const port = allPorts.value.find(p => p.external_port === externalPort)
  if (!port || !port.options) return ''
  const option = port.options.find(o => o.ip === ip)
  return option ? option.environment : ''
}

function getCandidates(externalPort) {
  const port = allPorts.value.find(p => p.external_port === externalPort)
  if (!port || !port.options) return []
  const currentIp = getRealTargetIp(externalPort)
  return port.options.filter(o => o.ip !== currentIp)
}

function hasCandidates(externalPort) {
  return getCandidates(externalPort).length > 0
}

async function confirmSave() {
  isSwitching.value = true
  try {
    await saveConfiguration()
  } finally {
    isSwitching.value = false
  }
}

onMounted(async () => {
  await initializeIPHeaders()
  await loadPortConfig()
  await loadPortStatus()
})
</script>

<style scoped>
:global(html, body) {
  height: 100%;
  margin: 0;
  padding: 0;
  background: #f5f6f8;
}

.app-container {
  min-height: 100vh;
  background: #f5f6f8;
  padding: 4vh 24px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.header {
  text-align: center;
  margin-bottom: 12px;
  flex-shrink: 0;
  position: relative;
}

.title {
  color: #1f2937;
  font-size: 28px;
  font-weight: 500;
  margin: 0;
  letter-spacing: 1px;
}

.header-actions {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  gap: 12px;
  align-items: center;
}

.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #6b7280;
  background: #ffffff;
  border: 1px solid #e2e4e8;
}

.theme-toggle:hover {
  color: #1f2937;
  background: #f0f1f3;
}

.doc-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #6b7280;
  background: #ffffff;
  border: 1px solid #e2e4e8;
  text-decoration: none;
}

.doc-link:hover {
  color: #1f2937;
  background: #f0f1f3;
}

.main-layout {
  display: flex;
  gap: 24px;
  max-width: 1600px;
  margin: 0 auto;
  flex: 1;
  min-height: 0;
  position: relative;
  padding: 0 24px;
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
  gap: 20px;
  height: 100%;
  overflow: hidden;
}

.main-workspace {
  display: flex;
  gap: 24px;
  flex: 1;
  min-height: 0;
}

.left-list-section,
.right-detail-section {
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e4e8;
  padding: 24px;
}

.left-list-section {
  width: 360px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.right-detail-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e4e8;
  flex-shrink: 0;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
}

.section-tip {
  font-size: 12px;
  color: #9ca3af;
}

.batch-action-section {
  margin-bottom: 12px;
  padding: 12px;
  background: #fff7ed;
  border: 1px solid #fed7aa;
  border-radius: 8px;
}

.section-title-small {
  font-size: 14px;
  font-weight: 500;
  color: #6b7280;
  margin-bottom: 12px;
}

.port-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.port-list-item {
  padding: 14px 16px;
  border-radius: 8px;
  border: 2px solid transparent;
  background: #f5f6f8;
  cursor: pointer;
  transition: all 0.15s ease;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.port-list-item:hover {
  background: #eef0f2;
  border-color: #d1d5db;
}

.port-list-item.selected {
  border-color: #3b82f6;
  background: #eff6ff;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.port-list-item.has-pending {
  border-left: 3px solid #f97316;
}

.item-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-desc {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
}

.port-address {
  font-size: 12px;
  color: #6b7280;
  font-family: 'Monaco', 'Menlo', monospace;
}

.port-pending {
  font-size: 12px;
  color: #f97316;
  font-family: 'Monaco', 'Menlo', monospace;
}

.item-status {
  flex-shrink: 0;
}

.detail-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.detail-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e2e4e8;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.detail-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.detail-address {
  font-size: 14px;
  color: #6b7280;
  font-family: 'Monaco', 'Menlo', monospace;
}

.current-status-section,
.pending-status-section,
.candidates-section {
  margin-bottom: 16px;
  flex-shrink: 0;
}

.current-status-card {
  padding: 16px;
  background: #f0fdf4;
  border: 1px solid #22c55e;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pending-status-card {
  padding: 16px;
  background: #fff7ed;
  border: 1px solid #f97316;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-ip {
  font-size: 16px;
  font-weight: 600;
  color: #166534;
  font-family: 'Monaco', 'Menlo', monospace;
}

.status-env {
  font-weight: 400;
  border: none;
  font-size: 12px;
  border-radius: 4px;
}

.candidates-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.candidates-list :deep(.el-radio-group) {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.candidate-item {
  padding: 14px 16px;
  border-radius: 8px;
  border: 2px solid transparent;
  background: #f5f6f8;
  transition: all 0.15s ease;
  cursor: pointer;
  box-sizing: border-box;
  display: block;
}

.candidate-item:hover {
  background: #eef0f2;
  border-color: #d1d5db;
}

.candidate-item.selected {
  border-color: #3b82f6;
  background: #eff6ff;
}

.candidate-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.candidate-ip {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.candidate-content :deep(.candidate-env) {
  font-weight: 400;
  border: none;
  font-size: 12px;
  border-radius: 4px;
  flex-shrink: 0;
}

.no-candidates-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.action-section {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid #e2e4e8;
  flex-shrink: 0;
}

.switch-button {
  width: 100%;
  padding: 16px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 8px;
}

.empty-detail {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logs-sidebar {
  position: absolute;
  left: 100%;
  top: 0;
  width: auto;
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e4e8;
  overflow: hidden;
  transition: all 0.3s ease;
  height: fit-content;
}

.status-section {
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e4e8;
  padding: 24px;
  text-align: center;
  flex-shrink: 0;
}

.status-grid-wrap {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  width: 100%;
}

.status-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 8px;
  background: #f5f6f8;
  border: 1px solid #e2e4e8;
  transition: all 0.2s ease;
  width: 100%;
  box-sizing: border-box;
}

.status-item.configured {
  border-color: #22c55e;
  background-color: #f0fdf4;
}

.status-item.not-configured {
  opacity: 0.6;
}

.status-port {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
  color: #1f2937;
  font-size: 14px;
  white-space: nowrap;
}

.status-arrow {
  color: #9ca3af;
  font-size: 14px;
}

.status-target {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
  color: #6b7280;
  font-size: 14px;
}

.status-indicator-wrapper {
  margin-left: auto;
}

.status-ok {
  color: #22c55e;
  font-size: 14px;
}

.status-empty {
  color: #d1d5db;
  font-size: 14px;
}

.dev-tag {
  background-color: #f0fdf4;
  color: #22c55e;
}

.zc-test-tag {
  background-color: #eff6ff;
  color: #3b82f6;
}

.dw-test-tag {
  background-color: #fefce8;
  color: #ca8a04;
}

.zc-hangshi-tag {
  background-color: #f0fdf4;
  color: #22c55e;
}

.zg-test-tag {
  background-color: #eff6ff;
  color: #3b82f6;
}

.port-list::-webkit-scrollbar,
.candidates-list::-webkit-scrollbar {
  width: 6px;
}

.port-list::-webkit-scrollbar-track,
.candidates-list::-webkit-scrollbar-track {
  background: transparent;
}

.port-list::-webkit-scrollbar-thumb,
.candidates-list::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}

.port-list::-webkit-scrollbar-thumb:hover,
.candidates-list::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

.dark-theme :global(html, body) {
  background: #0f1117;
}

.dark-theme {
  background: #0f1117;
}

.dark-theme .title {
  color: #e8eaed;
}

.dark-theme .theme-toggle {
  color: #7c8293;
  background: #1a1d26;
  border-color: #2a2d36;
}

.dark-theme .theme-toggle:hover {
  color: #e8eaed;
  background: #22252e;
}

.dark-theme .doc-link {
  color: #7c8293;
  background: #1a1d26;
  border-color: #2a2d36;
}

.dark-theme .doc-link:hover {
  color: #e8eaed;
  background: #22252e;
}

.dark-theme .left-list-section,
.dark-theme .right-detail-section {
  background: #1a1d26;
  border-color: #2a2d36;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.dark-theme .section-header {
  border-bottom-color: #2a2d36;
}

.dark-theme .section-title {
  color: #e8eaed;
}

.dark-theme .section-tip {
  color: #7c8293;
}

.dark-theme .batch-action-section {
  background: #291a14;
  border-color: #7c3b1f;
}

.dark-theme .section-title-small {
  color: #7c8293;
}

.dark-theme .port-list-item {
  background: #1e2129;
  border-color: #2a2d36;
}

.dark-theme .port-list-item:hover {
  background: #22252e;
  border-color: #3b3f4a;
}

.dark-theme .port-list-item.selected {
  border-color: #3b82f6;
  background: #1a2744;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.dark-theme .port-list-item.has-pending {
  border-left-color: #fb923c;
}

.dark-theme .port-desc {
  color: #e8eaed;
}

.dark-theme .port-address {
  color: #7c8293;
}

.dark-theme .port-pending {
  color: #fb923c;
}

.dark-theme .detail-title {
  color: #e8eaed;
}

.dark-theme .detail-address {
  color: #7c8293;
}

.dark-theme .current-status-card {
  background: #16351a;
  border-color: #4ade80;
}

.dark-theme .pending-status-card {
  background: #291a14;
  border-color: #fb923c;
}

.dark-theme .status-ip {
  color: #4ade80;
}

.dark-theme .candidate-item {
  background: #1e2129;
  border-color: #2a2d36;
}

.dark-theme .candidate-item:hover {
  background: #22252e;
  border-color: #3b3f4a;
}

.dark-theme .candidate-item.selected {
  border-color: #3b82f6;
  background: #1a2744;
}

.dark-theme .candidate-ip {
  color: #e8eaed;
}

.dark-theme .dev-tag {
  background-color: #16351a;
  color: #4ade80;
}

.dark-theme .zc-test-tag {
  background-color: #1a2744;
  color: #60a5fa;
}

.dark-theme .dw-test-tag {
  background-color: #2a2214;
  color: #facc15;
}

.dark-theme .zc-hangshi-tag {
  background-color: #16351a;
  color: #4ade80;
}

.dark-theme .zg-test-tag {
  background-color: #1a2744;
  color: #60a5fa;
}

.dark-theme .logs-sidebar {
  background: #1a1d26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border-color: #2a2d36;
}

.dark-theme .status-section {
  background: #1a1d26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border-color: #2a2d36;
}

.dark-theme .status-item {
  background: #1e2129;
  border-color: #2a2d36;
}

.dark-theme .status-item.configured {
  border-color: #4ade80;
  background-color: #1e2129;
}

.dark-theme .status-port {
  color: #e8eaed;
}

.dark-theme .status-arrow {
  color: #7c8293;
}

.dark-theme .status-target {
  color: #c0c4cc;
}

.dark-theme .status-ok {
  color: #4ade80;
}

.dark-theme .status-empty {
  color: #3b3f4a;
}

.dark-theme .action-section {
  border-top-color: #2a2d36;
}

.dark-theme .port-list::-webkit-scrollbar-thumb,
.dark-theme .candidates-list::-webkit-scrollbar-thumb {
  background: #3b3f4a;
}

.dark-theme .port-list::-webkit-scrollbar-thumb:hover,
.dark-theme .candidates-list::-webkit-scrollbar-thumb:hover {
  background: #4a4e57;
}
</style>
