import { ref, reactive, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

    // 检测本次变更是否引入重复映射
    const changedExtPorts = new Set(configs.map(c => c.external_port))

    const afterState = { ...originalMappings.value }
    configs.forEach(c => { afterState[`port_${c.external_port}`] = c.internal_ip })

    const extToInfo = {}
    expandedPortConfigs.forEach(p => {
      const allPorts = p.all_external_ports || [p.external_port]
      const idx = allPorts.indexOf(p.external_port)
      const label = allPorts.length > 1 ? `${p.description}-${idx + 1}` : p.description
      extToInfo[p.external_port] = { internalPort: p.internal_port, label, externalIP: p.external_ip }
    })

    const groups = {}
    Object.entries(afterState).forEach(([key, ip]) => {
      if (!ip) return
      const extPort = parseInt(key.replace('port_', ''))
      const info = extToInfo[extPort]
      if (!info) return
      const gKey = `${ip}|${info.internalPort}`
      if (!groups[gKey]) groups[gKey] = []
      groups[gKey].push({ extPort, ...info })
    })

    // 只保留至少有一个外网端口是本次变更的组
    const activeGroups = {}
    Object.entries(groups).forEach(([gKey, entries]) => {
      if (entries.length <= 1) return
      if (entries.some(e => changedExtPorts.has(e.extPort))) {
        activeGroups[gKey] = entries
      }
    })

    if (Object.keys(activeGroups).length > 0) {
      const tableRows = []
      Object.entries(activeGroups).forEach(([gKey, entries]) => {
        const [ip, intPortStr] = gKey.split('|')
        entries.sort((a, b) => a.extPort - b.extPort)
        entries.forEach((e, idx) => {
          tableRows.push(
            `<tr>` +
            `<td style="padding:6px 12px;border-right:1px solid #ebeef5;vertical-align:top;font-weight:500;color:#303133;">${idx === 0 ? `${ip}:${intPortStr}` : `<span style="color:#e6a23c;">&#8595; 重复映射</span>`}</td>` +
            `<td style="padding:6px 12px;line-height:1.7;">${e.label}<br><span style="color:#909399;font-size:12px;">${e.externalIP}:${e.extPort}</span></td>` +
            `</tr>`
          )
        })
      })

      const html = `
        <div style="text-align:left;">
          <p style="margin:0 0 10px;color:#303133;font-size:14px;">以下内网地址将会出现重复映射：</p>
          <table style="width:100%;border-collapse:collapse;border:1px solid #ebeef5;border-radius:4px;">
            <thead>
              <tr style="background:#f5f7fa;">
                <th style="padding:8px 12px;border-right:1px solid #ebeef5;text-align:left;font-weight:600;color:#606266;">内网地址</th>
                <th style="padding:8px 12px;text-align:left;font-weight:600;color:#606266;">外网映射目标</th>
              </tr>
            </thead>
            <tbody>${tableRows.join('')}</tbody>
          </table>
          <p style="margin:12px 0 0;color:#303133;font-size:14px;">是否确认继续配置？</p>
        </div>`

      try {
        await ElMessageBox.confirm(html, '重复映射提醒', {
          dangerouslyUseHTMLString: true,
          confirmButtonText: '确认继续',
          cancelButtonText: '取消',
          type: 'warning'
        })
      } catch {
        return
      }
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