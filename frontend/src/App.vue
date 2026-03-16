<template>
  <div class="app-container">
    <div class="header">
      <h1 class="title">星目路由器端口快切工具</h1>
    </div>
    
    <div class="content">
      <div class="port-cards">
        <!-- 端口 61002 -->
        <el-card class="port-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-content">
                <span class="port-desc">侦测无人机控制链路端口</span>
                <span class="port-title">117.149.14.2:61002</span>
              </div>
              <el-icon class="expand-icon"><ArrowDown /></el-icon>
            </div>
          </template>
          
          <div class="port-options">
            <el-radio-group v-model="portMappings.port61002" class="radio-group">
              <div class="radio-item">
                <el-radio value="192.168.1.109" class="custom-radio">
                  <span class="ip-text">192.168.1.109</span>
                  <el-tag class="env-tag dev-tag" size="small">Dev 环境</el-tag>
                </el-radio>
              </div>
              <div class="radio-item">
                <el-radio value="192.168.1.94" class="custom-radio">
                  <span class="ip-text">192.168.1.94</span>
                  <el-tag class="env-tag test-tag" size="small">Test 环境</el-tag>
                </el-radio>
              </div>
            </el-radio-group>
          </div>
        </el-card>

        <!-- 端口 61100 -->
        <el-card class="port-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-content">
                <span class="port-desc">侦测无人机数据链路端口</span>
                <span class="port-title">117.149.14.2:61100</span>
              </div>
              <el-icon class="expand-icon"><ArrowDown /></el-icon>
            </div>
          </template>
          
          <div class="port-options">
            <el-radio-group v-model="portMappings.port61100" class="radio-group">
              <div class="radio-item">
                <el-radio value="192.168.1.109" class="custom-radio">
                  <span class="ip-text">192.168.1.109</span>
                  <el-tag class="env-tag dev-tag" size="small">Dev 环境</el-tag>
                </el-radio>
              </div>
              <div class="radio-item">
                <el-radio value="192.168.1.94" class="custom-radio">
                  <span class="ip-text">192.168.1.94</span>
                  <el-tag class="env-tag test-tag" size="small">Test 环境</el-tag>
                </el-radio>
              </div>
            </el-radio-group>
          </div>
        </el-card>

        <!-- 端口 48099 -->
        <el-card class="port-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-content">
                <span class="port-desc">东吴后端端口</span>
                <span class="port-title">117.149.14.2:48099</span>
              </div>
              <el-icon class="expand-icon"><ArrowDown /></el-icon>
            </div>
          </template>
          
          <div class="port-options">
            <el-radio-group v-model="portMappings.port48099" class="radio-group">
              <div class="radio-item">
                <el-radio value="192.168.1.109" class="custom-radio">
                  <span class="ip-text">192.168.1.109</span>
                  <el-tag class="env-tag dev-tag" size="small">Dev 环境</el-tag>
                </el-radio>
              </div>
              <div class="radio-item">
                <el-radio value="192.168.1.116" class="custom-radio">
                  <span class="ip-text">192.168.1.116</span>
                  <el-tag class="env-tag test-tag" size="small">Test 环境</el-tag>
                </el-radio>
              </div>
            </el-radio-group>
          </div>
        </el-card>
      </div>

      <!-- 当前映射状态 -->
      <el-card class="status-card" shadow="hover">
        <div class="status-content">
          <div class="status-item" :class="{ active: portMappings.port61002 }">
            <el-icon class="status-icon" :class="{ active: portMappings.port61002 }">
              <CircleCheckFilled v-if="portMappings.port61002" />
              <CircleFilled v-else />
            </el-icon>
            <span class="status-text">
              61002 → {{ portMappings.port61002 || '未选择' }}
            </span>
          </div>
          
          <div class="status-item" :class="{ active: portMappings.port61100 }">
            <el-icon class="status-icon" :class="{ active: portMappings.port61100 }">
              <CircleCheckFilled v-if="portMappings.port61100" />
              <CircleFilled v-else />
            </el-icon>
            <span class="status-text">
              61100 → {{ portMappings.port61100 || '未选择' }}
            </span>
          </div>
          
          <div class="status-item" :class="{ active: portMappings.port48099 }">
            <el-icon class="status-icon" :class="{ active: portMappings.port48099 }">
              <CircleCheckFilled v-if="portMappings.port48099" />
              <CircleFilled v-else />
            </el-icon>
            <span class="status-text">
              48099 → {{ portMappings.port48099 || '未选择' }}
            </span>
          </div>
        </div>
      </el-card>

      <!-- 保存按钮 -->
      <div class="action-area">
        <el-button 
          type="primary" 
          size="large" 
          class="save-button"
          @click="saveConfiguration"
          :loading="saving"
        >
          <el-icon><DocumentAdd /></el-icon>
          保存并应用配置
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

// 端口映射配置
const portMappings = reactive({
  port61002: '192.168.1.109', // 默认选择开发环境
  port61100: '192.168.1.109', // 默认选择开发环境
  port48099: '192.168.1.116'  // 默认选择测试环境
})

const saving = ref(false)

// 保存配置
const saveConfiguration = async () => {
  // 检查是否所有端口都已配置
  if (!portMappings.port61002 || !portMappings.port61100 || !portMappings.port48099) {
    ElMessage.warning('请为所有端口选择映射地址')
    return
  }

  saving.value = true
  
  try {
    // 这里后续会调用后端API保存配置
    await new Promise(resolve => setTimeout(resolve, 1500)) // 模拟API调用
    
    ElMessage.success('配置保存成功！')
    console.log('保存的配置:', portMappings)
  } catch (error) {
    ElMessage.error('保存配置失败，请重试')
    console.error('保存失败:', error)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.app-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.header {
  text-align: center;
  margin-bottom: 40px;
}

.title {
  color: white;
  font-size: 32px;
  font-weight: 600;
  margin: 0;
  text-shadow: 0 2px 4px rgba(0,0,0,0.3);
}

.content {
  max-width: 1200px;
  margin: 0 auto;
}

.port-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  margin-bottom: 32px;
}

.port-card {
  border-radius: 16px;
  border: none;
  background: white;
  transition: all 0.3s ease;
}

.port-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px rgba(0,0,0,0.15);
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
  gap: 4px;
  flex: 1;
}

.port-desc {
  font-size: 18px;
  color: #2c3e50;
  font-weight: 700;
  text-align: center;
}

.port-title {
  font-size: 12px;
  font-weight: 400;
  color: #909399;
  text-align: center;
}

.expand-icon {
  color: #909399;
  font-size: 16px;
}

.port-options {
  padding: 8px 0;
}

.radio-group {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.radio-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  transition: background-color 0.2s ease;
}

.radio-item:hover {
  background-color: #f8f9fa;
}

.custom-radio {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 0;
}

.ip-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 16px;
  font-weight: 500;
  color: #2c3e50;
}

.env-tag {
  font-weight: 500;
  border: none;
}

.dev-tag {
  background-color: #e8f5e8;
  color: #52c41a;
}

.test-tag {
  background-color: #e6f4ff;
  color: #1890ff;
}

.status-card {
  border-radius: 16px;
  border: none;
  background: white;
  margin-bottom: 32px;
}

.status-content {
  display: flex;
  justify-content: space-around;
  align-items: center;
  padding: 16px 0;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.status-item.active {
  background-color: #f0f9ff;
}

.status-icon {
  font-size: 16px;
  color: #d1d5db;
}

.status-icon.active {
  color: #10b981;
}

.status-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.action-area {
  text-align: center;
}

.save-button {
  padding: 16px 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
  transition: all 0.3s ease;
}

.save-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(59, 130, 246, 0.5);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .app-container {
    padding: 16px;
  }
  
  .title {
    font-size: 24px;
  }
  
  .port-cards {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .status-content {
    flex-direction: column;
    gap: 12px;
  }
  
  .save-button {
    width: 100%;
    padding: 16px 24px;
  }
}
</style>