import { ref, reactive, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchPortConfig, fetchPortStatus, applyConfig } from '../api'

export function usePortConfig() {
  const portConfigs = ref([])
  const portMappings = ref({})
  const originalMappings = ref({})
  const expandedState = reactive({})
  const saving = ref(false)
  const loading = ref(false)
  const hasChanges = ref(false)
  const operationLogsRef = ref(null)

  watch(portMappings, () => {
    hasChanges.value = false
    for (const key in portMappings.value) {
      if (portMappings.value[key] !== originalMappings.value[key]) {
        hasChanges.value = true
        break
      }
    }
  }, { deep: true })

  const togglePortCard = (key) => {
    expandedState[key] = !expandedState[key]
  }

  const getExpandedPorts = () => {
    return portConfigs.value.flatMap(p =>
      p.all_external_ports && p.all_external_ports.length > 0
        ? p.all_external_ports.map(extPort => ({ ...p, external_port: extPort }))
        : [{ ...p }]
    )
  }

  const getPortIndex = (port) => {
    if (!portConfigs.value || !Array.isArray(portConfigs.value)) return 1
    const originalPortConfig = portConfigs.value.find(p => p.internal_port === port.internal_port)
    if (originalPortConfig && originalPortConfig.all_external_ports) {
      const index = originalPortConfig.all_external_ports.indexOf(port.external_port)
      return index >= 0 ? index + 1 : 1
    }
    return 1
  }

  const hasMultiplePorts = (port) => {
    if (!portConfigs.value || !Array.isArray(portConfigs.value)) return false
    const originalPortConfig = portConfigs.value.find(p => p.internal_port === port.internal_port)
    return originalPortConfig && originalPortConfig.all_external_ports && originalPortConfig.all_external_ports.length > 1
  }

  const loadPortConfig = async () => {
    try {
      const sortedPorts = await fetchPortConfig()
      portConfigs.value = sortedPorts

      const expandedPortConfigs = getExpandedPorts()
      const newPortMappings = {}
      const newOriginalMappings = {}
      expandedPortConfigs.forEach(port => {
        const key = `port_${port.external_port}`
        newPortMappings[key] = ''
        newOriginalMappings[key] = ''
      })
      portMappings.value = newPortMappings
      originalMappings.value = newOriginalMappings
    } catch (error) {
      console.error('获取端口配置失败:', error)
      ElMessage.error('获取端口配置失败，请检查后端服务')
    }
  }

  const loadPortStatus = async () => {
    loading.value = true
    try {
      const status = await fetchPortStatus()
      const newPortMappings = { ...portMappings.value }
      const newOriginalMappings = { ...originalMappings.value }
      for (const key in status) {
        newPortMappings[key] = status[key] || ''
        newOriginalMappings[key] = status[key] || ''
      }
      portMappings.value = newPortMappings
      originalMappings.value = newOriginalMappings
      await nextTick()
    } catch (error) {
      console.error('获取端口状态失败:', error)
      ElMessage.error('获取端口状态失败，请检查后端服务')
    } finally {
      loading.value = false
    }
  }

  const saveConfiguration = async () => {
    const expandedPortConfigs = getExpandedPorts()

    const configs = []
    expandedPortConfigs.forEach(port => {
      const key = `port_${port.external_port}`
      if (portMappings.value[key] && portMappings.value[key] !== originalMappings.value[key]) {
        configs.push({
          internal_port: port.internal_port,
          internal_ip: portMappings.value[key],
          external_port: port.external_port
        })
      }
    })

    if (configs.length === 0) {
      ElMessage.info('没有配置变更需要保存')
      return
    }

    saving.value = true
    try {
      await applyConfig(configs)
      await loadPortStatus()
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

  return {
    portConfigs,
    portMappings,
    originalMappings,
    expandedState,
    saving,
    loading,
    hasChanges,
    operationLogsRef,
    togglePortCard,
    getExpandedPorts,
    getPortIndex,
    hasMultiplePorts,
    loadPortConfig,
    loadPortStatus,
    saveConfiguration
  }
}