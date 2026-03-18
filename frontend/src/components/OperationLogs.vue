<template>
  <div class="operation-logs-card" :class="{ 'collapsed': isCollapsed, 'expanded': !isCollapsed }">
    <!-- 折叠状态的小箭头指引 -->
    <div v-if="isCollapsed" class="collapsed-indicator" @click="toggleCollapse">
      <el-icon class="arrow-icon">
        <ArrowLeft />
      </el-icon>
      <span class="indicator-text">日志</span>
    </div>
    
    <!-- 展开状态的完整日志 -->
    <div v-else class="expanded-content">
      <div class="card-header">
        <h4 class="card-title">操作日志</h4>
        <div class="header-actions">
          <el-button 
            type="text" 
            :icon="RefreshRight" 
            @click="refreshLogs"
            :loading="loading"
            class="refresh-btn"
            size="small"
          />
          <el-button 
            type="text" 
            :icon="ArrowRight" 
            @click="toggleCollapse"
            class="collapse-btn"
            size="small"
          />
        </div>
      </div>
      
      <div class="card-content" v-loading="loading">
        <div v-if="error" class="error-state">
          <span class="error-text">加载失败</span>
          <el-button type="text" @click="refreshLogs" size="small">重试</el-button>
        </div>
        
        <div v-else-if="logs.length === 0" class="empty-state">
          <span class="empty-text">暂无记录</span>
        </div>
        
        <div v-else class="logs-list">
          <div 
            v-for="log in displayLogs" 
            :key="log.id"
            class="log-entry"
            :class="{ 'success': log.status === '成功', 'failed': log.status === '失败' }"
          >
            <div class="log-header">
              <span class="log-time">{{ log.timestamp }}</span>
              <el-icon 
                class="status-icon" 
                :class="{ 'success': log.status === '成功', 'failed': log.status === '失败' }"
              >
                <CircleCheckFilled v-if="log.status === '成功'" />
                <CircleCloseFilled v-else />
              </el-icon>
            </div>
            
            <div class="log-details">
              <div class="log-field">
                <span class="field-label">操作类型：</span>
                <span class="field-value">{{ log.operation || '端口映射切换' }}</span>
              </div>
              
              <div class="log-field" v-if="log.source_port_ip && log.target_port_ip">
                <span class="field-label">端口：</span>
                <span class="field-value">{{ getPortFromMapping(log.source_port_ip) }}</span>
              </div>
              
              <div class="log-field">
                <span class="field-label">操作者来源：</span>
                <span class="field-value">{{ log.operator_ip }}</span>
              </div>
              
              <div class="log-field" v-if="log.source_port_ip && log.target_port_ip">
                <span class="field-label">原映射：</span>
                <span class="field-value">{{ getIPFromMapping(log.source_port_ip) }}</span>
              </div>
              
              <div class="log-field" v-if="log.source_port_ip && log.target_port_ip">
                <span class="field-label">新映射：</span>
                <span class="field-value">{{ getIPFromMapping(log.target_port_ip) }}</span>
              </div>
              
              <div class="log-field">
                <span class="field-label">结果：</span>
                <span class="field-value" :class="{ 'success-text': log.status === '成功', 'failed-text': log.status === '失败' }">
                  {{ log.status }}
                </span>
              </div>
            </div>
          </div>
        </div>
        
        <div v-if="logs.length > 5" class="more-logs">
          <span class="more-text">共 {{ logs.length }} 条记录</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  RefreshRight, 
  CircleCheckFilled, 
  CircleCloseFilled,
  ArrowLeft,
  ArrowRight
} from '@element-plus/icons-vue'
import axios from 'axios'

// API基础URL
const getApiBaseUrl = () => {
  const hostname = window.location.hostname
  const port = '8080'
  return `http://${hostname}:${port}`
}

const API_BASE_URL = getApiBaseUrl()

// 响应式数据
const logs = ref([])
const loading = ref(false)
const error = ref('')
const isCollapsed = ref(true) // 默认折叠状态

// 只显示最近5条日志
const displayLogs = computed(() => {
  return logs.value.slice(0, 5)
})

// 切换折叠/展开状态
const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}

// 从映射字符串中提取端口号
const getPortFromMapping = (portIP) => {
  if (!portIP) return ''
  const [port] = portIP.split('_')
  return port
}

// 从映射字符串中提取IP地址
const getIPFromMapping = (portIP) => {
  if (!portIP) return ''
  const [, ip] = portIP.split('_')
  return ip
}

// 获取操作日志
const fetchLogs = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const response = await axios.get(`${API_BASE_URL}/api/operation-logs`)
    logs.value = response.data.logs || []
  } catch (err) {
    console.error('获取操作日志失败:', err)
    error.value = '获取失败'
  } finally {
    loading.value = false
  }
}

// 刷新日志
const refreshLogs = async () => {
  await fetchLogs()
  if (!error.value) {
    ElMessage.success('已刷新')
  }
}

// 组件挂载时获取日志
onMounted(() => {
  fetchLogs()
})

// 暴露刷新方法给父组件
defineExpose({
  refreshLogs
})
</script>

<style scoped>
.operation-logs-card {
  width: 100%;
  background: transparent;
  border: none;
  box-shadow: none;
  display: flex;
  flex-direction: column;
  transition: all 0.3s ease;
}

/* 折叠状态 */
.operation-logs-card.collapsed {
  width: 60px;
  height: 120px;
}

/* 展开状态 */
.operation-logs-card.expanded {
  width: 350px;
  height: 60vh;
  max-height: 60vh;
  min-height: 200px;
}

/* 折叠状态的指引箭头 */
.collapsed-indicator {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.collapsed-indicator:hover {
  background: rgba(255, 255, 255, 1);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
  transform: translateX(-2px);
}

.arrow-icon {
  font-size: 20px;
  color: #667eea;
  margin-bottom: 8px;
}

.indicator-text {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
  writing-mode: vertical-rl;
  text-orientation: mixed;
}

/* 展开状态的内容 */
.expanded-content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  background: rgba(248, 250, 252, 0.8);
  flex-shrink: 0;
}

.card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #64748b;
}

.header-actions {
  display: flex;
  gap: 4px;
}

.refresh-btn,
.collapse-btn {
  padding: 4px;
  color: #94a3b8;
  font-size: 14px;
}

.refresh-btn:hover,
.collapse-btn:hover {
  color: #409eff;
}

.card-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0;
  min-height: 0;
}

.error-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 120px;
  color: #94a3b8;
}

.error-text,
.empty-text {
  font-size: 13px;
  margin-bottom: 8px;
}

.logs-list {
  padding: 0;
}

.log-entry {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}

.log-entry:last-child {
  border-bottom: none;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.log-time {
  font-size: 12px;
  color: #94a3b8;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
}

.status-icon {
  font-size: 14px;
}

.status-icon.success {
  color: #52c41a;
}

.status-icon.failed {
  color: #ff4d4f;
}

.log-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-field {
  display: flex;
  align-items: center;
  font-size: 12px;
  line-height: 1.4;
}

.field-label {
  color: #94a3b8;
  min-width: 60px;
  font-weight: 500;
}

.field-value {
  color: #475569;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
}

.success-text {
  color: #52c41a;
}

.failed-text {
  color: #ff4d4f;
}

.more-logs {
  text-align: center;
  padding: 8px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.05);
  background: rgba(248, 250, 252, 0.8);
  flex-shrink: 0;
}

.more-text {
  font-size: 12px;
  color: #94a3b8;
}

/* 滚动条样式 */
.card-content::-webkit-scrollbar {
  width: 3px;
}

.card-content::-webkit-scrollbar-track {
  background: transparent;
}

.card-content::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.3);
  border-radius: 2px;
}

.card-content::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.5);
}
</style>