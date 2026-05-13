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
      <div class="status-section">
        <div class="section-header">
          <h2 class="section-title">当前端口映射状态</h2>
        </div>

        <div class="status-grid-wrap">
          <div
            v-for="port in portConfigs.flatMap(p => 
              p.all_external_ports && p.all_external_ports.length > 0 
                ? p.all_external_ports.map(extPort => ({...p, external_port: extPort})) 
                : [{...p}]
            )"
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
          <div class="grid-container">
            <div
              v-for="port in portConfigs.flatMap(p => 
                p.all_external_ports && p.all_external_ports.length > 0 
                  ? p.all_external_ports.map(extPort => ({...p, external_port: extPort})) 
                  : [{...p}]
              )"
              :key="`config-${port.internal_port}-${port.external_port}`"
              class="grid-item"
            >
              <div class="port-card">
                <div 
                  class="card-header" 
                  @click="togglePortCard(port.internal_port + '-' + port.external_port)"
                >
                  <div class="header-content">
                    <span class="port-desc">{{ port.description }}<span v-if="hasMultiplePorts(port)"> {{ getPortIndex(port) }}</span></span>
                    <span class="port-title">{{ port.external_ip }}:{{ port.external_port }}</span>
                  </div>
                  <el-icon class="expand-icon" :class="{ 'rotated': expandedState[port.internal_port + '-' + port.external_port] }">
                    <ArrowDown />
                  </el-icon>
                </div>
                
                <div 
                  v-show="expandedState[port.internal_port + '-' + port.external_port]"
                  class="port-options"
                >
                  <el-radio-group v-model="portMappings[`port_${port.external_port}`]" class="radio-group">
                    <div
                      v-for="option in port.options"
                      :key="option.ip"
                      class="radio-item"
                    >
                      <el-radio :value="option.ip" class="custom-radio">
                        <span class="ip-text">{{ option.ip }}:{{ port.internal_port }}</span>
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
              </div>
            </div>
          </div>
        </div>

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

      <div class="logs-sidebar">
        <OperationLogs ref="operationLogsRef" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { ArrowDown, ArrowRight, CircleCheckFilled, DocumentAdd, Clock, Sunny, Moon, Document } from '@element-plus/icons-vue'
import { useTheme } from '../composables/useTheme'
import { usePortConfig } from '../composables/usePortConfig'
import { initializeIPHeaders } from '../api'
import OperationLogs from '../components/OperationLogs.vue'

const { isDarkTheme, toggleTheme } = useTheme()
const {
  portConfigs,
  portMappings,
  originalMappings,
  expandedState,
  saving,
  loading,
  hasChanges,
  operationLogsRef,
  togglePortCard,
  getPortIndex,
  hasMultiplePorts,
  loadPortConfig,
  loadPortStatus,
  saveConfiguration
} = usePortConfig()

const docUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/readme.html`

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
  gap: 16px;
  max-width: 1100px;
  margin: 0 auto;
  flex: 1;
  min-height: 0;
  position: relative;
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

.config-section {
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e4e8;
  padding: 12px;
  flex-shrink: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e4e8;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
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
  color: #22c55e;
}

.status-indicator.pending {
  color: #eab308;
}

.status-text {
  font-size: 12px;
  font-weight: 400;
  color: #6b7280;
}

.port-cards {
  width: 100%;
}

.grid-container {
  column-count: 2;
  column-gap: 16px;
}

.grid-item {
  break-inside: avoid;
  margin-bottom: 16px;
  display: inline-block;
  width: 100%;
}

.port-card {
  border: 1px solid #e2e4e8;
  border-radius: 12px;
  transition: all 0.2s ease;
  cursor: pointer;
  min-height: 80px;
  display: flex;
  flex-direction: column;
  background: #ffffff;
}

.port-card:hover {
  border-color: #d1d5db;
  transform: translateY(-1px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  cursor: pointer;
  border-radius: 11px;
  transition: background-color 0.2s ease;
}

.card-header:hover {
  background-color: #f5f6f8;
}

.header-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.port-desc {
  font-size: 15px;
  color: #1f2937;
  font-weight: 500;
  text-align: center;
}

.port-title {
  font-size: 13px;
  font-weight: 400;
  color: #6b7280;
  text-align: center;
  font-family: 'Monaco', 'Menlo', monospace;
}

.expand-icon {
  color: #9ca3af;
  font-size: 12px;
  cursor: pointer;
  transition: transform 0.3s ease;
}

.expand-icon.rotated {
  transform: rotate(180deg);
}

.port-options {
  padding: 12px 0;
  flex: 1;
  overflow-y: auto;
  min-height: 0;
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
  padding: 10px 14px;
  border-radius: 8px;
  transition: all 0.15s ease;
  border: 1px solid transparent;
  background: #f5f6f8;
}

.radio-item:hover {
  background-color: #eef0f2;
  border-color: #e2e4e8;
}

.custom-radio {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 0;
}

.custom-radio.is-checked .radio-item {
  background-color: #eff6ff;
  border-color: #3b82f6;
}

.custom-radio.is-checked .ip-text {
  color: #3b82f6;
  font-weight: 600;
}

.ip-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
}

.env-tag {
  font-weight: 400;
  border: none;
  font-size: 12px;
  border-radius: 4px;
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

.status-section {
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #e2e4e8;
  padding: 16px;
  text-align: center;
  flex-shrink: 0;
}

.status-grid-wrap {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 8px;
  background: #f5f6f8;
  border: 1px solid #e2e4e8;
  transition: all 0.2s ease;
  width: fit-content;
  margin: 0 auto;
}

.status-item.current {
  border-color: #22c55e;
  background-color: #f0fdf4;
}

.status-item.changed {
  border-color: #eab308;
  background-color: #fefce8;
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

.status-pending {
  color: #eab308;
  font-size: 14px;
}

.action-area {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e2e4e8;
}

.save-button {
  padding: 14px 40px;
  font-size: 15px;
  font-weight: 500;
  border-radius: 8px;
  border: 1px solid transparent;
  transition: all 0.2s ease;
  min-width: 200px;
  color: white;
}

.save-button.has-changes {
  background: #3b82f6;
  border-color: #3b82f6;
  color: white;
}

.save-button.has-changes:hover {
  background: #2563eb;
  transform: translateY(-1px);
}

.save-button.no-changes {
  background: #22c55e;
  border-color: #22c55e;
  color: #ffffff;
}

.save-button.no-changes:hover {
  background: #16a34a;
}

.port-options::-webkit-scrollbar {
  width: 3px;
}

.port-options::-webkit-scrollbar-track {
  background: transparent;
}

.port-options::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 2px;
}

.port-options::-webkit-scrollbar-thumb:hover {
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

.dark-theme .logs-sidebar {
  background: #1a1d26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border-color: #2a2d36;
}

.dark-theme .config-section {
  background: #1a1d26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border-color: #2a2d36;
}

.dark-theme .section-header {
  border-bottom-color: #2a2d36;
}

.dark-theme .section-title {
  color: #e8eaed;
}

.dark-theme .status-indicator.synced {
  color: #4ade80;
}

.dark-theme .status-indicator.pending {
  color: #facc15;
}

.dark-theme .status-text {
  color: #7c8293;
}

.dark-theme .port-card {
  border-color: #2a2d36;
  background: #1e2129;
}

.dark-theme .port-card:hover {
  border-color: #3b3f4a;
}

.dark-theme .card-header:hover {
  background-color: #22252e;
}

.dark-theme .port-desc {
  color: #e8eaed;
}

.dark-theme .port-title {
  color: #7c8293;
}

.dark-theme .expand-icon {
  color: #7c8293;
}

.dark-theme .radio-item {
  background: #1a1d26;
}

.dark-theme .radio-item:hover {
  background-color: #22252e;
  border-color: #2a2d36;
}

.dark-theme .custom-radio.is-checked .radio-item {
  background-color: #262930;
  border-color: #3b82f6;
}

.dark-theme .ip-text {
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

.dark-theme .status-section {
  background: #1a1d26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  border-color: #2a2d36;
}

.dark-theme .status-item {
  background: #1e2129;
  border-color: #2a2d36;
}

.dark-theme .status-item.current {
  border-color: #4ade80;
  background-color: #1e2129;
}

.dark-theme .status-item.changed {
  border-color: #facc15;
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

.dark-theme .status-pending {
  color: #facc15;
}

.dark-theme .action-area {
  border-top-color: #2a2d36;
}

.dark-theme .save-button.no-changes {
  color: #0f1117;
}

.dark-theme .port-options::-webkit-scrollbar-thumb {
  background: #3b3f4a;
}

.dark-theme .port-options::-webkit-scrollbar-thumb:hover {
  background: #4a4e57;
}
</style>