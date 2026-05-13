import axios from 'axios'

export const getApiBaseUrl = () => {
  if (import.meta.env.DEV) {
    const hostname = window.location.hostname
    return `http://${hostname}:8080`
  }
  return ''
}

export const getClientInternalIP = () => {
  const hostname = window.location.hostname
  if (hostname.match(/^192\.168\.|^10\.|^172\.(1[6-9]|2[0-9]|3[0-1])\./)) {
    return hostname
  }
  return 'unknown'
}

export const getClientExternalIP = async () => {
  const internalIP = getClientInternalIP()
  const fingerprint = btoa(
    navigator.userAgent +
    screen.width + 'x' + screen.height +
    new Date().getTimezoneOffset()
  ).substr(0, 8)
  return `${internalIP}-${fingerprint}`
}

export const initializeIPHeaders = async () => {
  const clientInternalIP = getClientInternalIP()
  const clientExternalIP = await getClientExternalIP()
  axios.defaults.headers.common['X-Client-Internal-IP'] = clientInternalIP
  axios.defaults.headers.common['X-Client-External-IP'] = clientExternalIP
}

export const fetchPortConfig = async () => {
  const API_BASE_URL = getApiBaseUrl()
  const response = await axios.get(`${API_BASE_URL}/api/port-config`)
  return response.data.ports.map(port => ({
    ...port,
    options: [...port.options].sort((a, b) => {
      const ipA = a.ip.split('.').map(Number)
      const ipB = b.ip.split('.').map(Number)
      for (let i = 0; i < 4; i++) {
        if (ipA[i] !== ipB[i]) return ipA[i] - ipB[i]
      }
      return 0
    })
  }))
}

export const fetchPortStatus = async () => {
  const API_BASE_URL = getApiBaseUrl()
  const response = await axios.get(`${API_BASE_URL}/api/port-status`)
  return response.data
}

export const applyConfig = async (configs) => {
  const API_BASE_URL = getApiBaseUrl()
  const response = await axios.post(`${API_BASE_URL}/api/apply-config`, { configs })
  return response.data
}