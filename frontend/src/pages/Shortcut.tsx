import { useState, useEffect } from 'react'
import { Key, Trash2, Copy, MessageSquare, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import Header from '../components/Header'
import { api } from '../api'
import styles from './Shortcut.module.css'

interface APIKey {
  id: number
  name: string
  is_active: number
  last_used_at: string | null
  created_at: string
}

interface SMSLog {
  id: number
  raw_text: string
  parsed_card_name: string | null
  parsed_amount: number | null
  parsed_description: string | null
  parsed_date: string | null
  transaction_id: number | null
  status: string
  error_message: string | null
  created_at: string
}

export default function Shortcut() {
  const [apiKeys, setAPIKeys] = useState<APIKey[]>([])
  const [smsLogs, setSMSLogs] = useState<SMSLog[]>([])
  const [newKeyName, setNewKeyName] = useState('')
  const [generatedKey, setGeneratedKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<'keys' | 'logs'>('keys')
  
  useEffect(() => {
    loadData()
  }, [])
  
  async function loadData() {
    setLoading(true)
    try {
      const [keys, logs] = await Promise.all([
        api.getAPIKeys(),
        api.getSMSLogs(50),
      ])
      setAPIKeys(keys)
      setSMSLogs(logs)
    } catch (e) {
      console.error('Failed to load data:', e)
    } finally {
      setLoading(false)
    }
  }
  
  async function createKey() {
    if (!newKeyName.trim()) return
    try {
      const result = await api.createAPIKey(newKeyName.trim())
      setGeneratedKey(result.api_key)
      setNewKeyName('')
      loadData()
    } catch (e) {
      console.error('Failed to create API key:', e)
      alert('API 키 생성에 실패했습니다.')
    }
  }
  
  async function deleteKey(id: number) {
    if (!confirm('이 API 키를 비활성화하시겠습니까?')) return
    try {
      await api.deleteAPIKey(id)
      loadData()
    } catch (e) {
      console.error('Failed to delete API key:', e)
    }
  }
  
  async function deleteSMSLog(id: number) {
    if (!confirm('이 로그를 삭제하시겠습니까?')) return
    try {
      await api.deleteSMSLog(id)
      loadData()
    } catch (e) {
      console.error('Failed to delete SMS log:', e)
    }
  }
  
  async function deleteAllSMSLogs() {
    if (!confirm('모든 SMS 로그를 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.')) return
    try {
      await api.deleteAllSMSLogs()
      loadData()
    } catch (e) {
      console.error('Failed to delete all SMS logs:', e)
    }
  }
  
  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  
  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString('ko-KR')
  }
  
  function formatAmount(amount: number | null) {
    if (amount === null) return '-'
    return amount.toLocaleString() + '원'
  }
  
  const apiEndpoint = `${window.location.origin}/api/shortcut/message`
  
  return (
    <div className={styles.page}>
      <Header title="단축어 연동" />
      
      <div className={styles.tabs}>
        <button 
          className={`${styles.tab} ${tab === 'keys' ? styles.active : ''}`}
          onClick={() => setTab('keys')}
        >
          <Key size={16} /> API 키 관리
        </button>
        <button 
          className={`${styles.tab} ${tab === 'logs' ? styles.active : ''}`}
          onClick={() => setTab('logs')}
        >
          <MessageSquare size={16} /> 수신 로그
        </button>
      </div>
      
      {tab === 'keys' && (
        <div className={styles.content}>
          <div className={styles.section}>
            <h3>아이폰 단축어 설정 가이드</h3>
            <div className={styles.guide}>
              <ol>
                <li>단축어 앱에서 <strong>자동화</strong> 탭 → <strong>메시지</strong> 트리거 선택</li>
                <li><strong>URL 콘텐츠 가져오기</strong> 동작 추가</li>
                <li>URL: <code>{apiEndpoint}</code></li>
                <li>방법: <strong>POST</strong></li>
                <li>헤더에 <code>X-API-Key</code> 추가 (아래에서 생성)</li>
                <li>본문: JSON, <code>text</code> 키에 메시지 내용 변수</li>
              </ol>
            </div>
          </div>
          
          <div className={styles.section}>
            <h3>API 엔드포인트</h3>
            <div className={styles.endpointBox}>
              <code>{apiEndpoint}</code>
              <button onClick={() => copyToClipboard(apiEndpoint)} className={styles.copyBtn}>
                <Copy size={16} />
              </button>
            </div>
          </div>
          
          <div className={styles.section}>
            <h3>새 API 키 생성</h3>
            <div className={styles.createForm}>
              <input
                type="text"
                placeholder="키 이름 (예: iPhone 단축어)"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                className={styles.input}
              />
              <button onClick={createKey} className={styles.createBtn}>
                생성
              </button>
            </div>
            
            {generatedKey && (
              <div className={styles.generatedKey}>
                <div className={styles.keyWarning}>
                  ⚠️ 이 키는 다시 표시되지 않습니다. 지금 복사하세요!
                </div>
                <div className={styles.keyBox}>
                  <code>{generatedKey}</code>
                  <button 
                    onClick={() => copyToClipboard(generatedKey)} 
                    className={styles.copyBtn}
                  >
                    {copied ? <CheckCircle size={16} /> : <Copy size={16} />}
                  </button>
                </div>
              </div>
            )}
          </div>
          
          <div className={styles.section}>
            <h3>API 키 목록</h3>
            {loading ? (
              <div className={styles.loading}>로딩 중...</div>
            ) : apiKeys.length === 0 ? (
              <div className={styles.empty}>API 키가 없습니다.</div>
            ) : (
              <div className={styles.keyList}>
                {apiKeys.map(key => (
                  <div key={key.id} className={`${styles.keyItem} ${key.is_active ? '' : styles.inactive}`}>
                    <div className={styles.keyInfo}>
                      <div className={styles.keyName}>
                        <Key size={16} /> {key.name}
                      </div>
                      <div className={styles.keyMeta}>
                        생성: {formatDate(key.created_at)}
                        {key.last_used_at && <> · 마지막 사용: {formatDate(key.last_used_at)}</>}
                      </div>
                    </div>
                    {key.is_active ? (
                      <button onClick={() => deleteKey(key.id)} className={styles.deleteBtn}>
                        <Trash2 size={16} />
                      </button>
                    ) : (
                      <span className={styles.inactiveLabel}>비활성</span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
      
      {tab === 'logs' && (
        <div className={styles.content}>
          <div className={styles.section}>
            <div className={styles.sectionHeader}>
              <h3>최근 SMS 수신 로그</h3>
              {smsLogs.length > 0 && (
                <button onClick={deleteAllSMSLogs} className={styles.deleteAllBtn}>
                  <Trash2 size={14} /> 전체 삭제
                </button>
              )}
            </div>
            {loading ? (
              <div className={styles.loading}>로딩 중...</div>
            ) : smsLogs.length === 0 ? (
              <div className={styles.empty}>수신된 SMS가 없습니다.</div>
            ) : (
              <div className={styles.logList}>
                {smsLogs.map(log => (
                  <div key={log.id} className={styles.logItem}>
                    <div className={styles.logHeader}>
                      <span className={`${styles.status} ${styles[log.status]}`}>
                        {log.status === 'saved' && <CheckCircle size={14} />}
                        {log.status === 'error' && <XCircle size={14} />}
                        {log.status === 'received' && <AlertCircle size={14} />}
                        {log.status}
                      </span>
                      <div className={styles.logActions}>
                        <span className={styles.logTime}>{formatDate(log.created_at)}</span>
                        <button onClick={() => deleteSMSLog(log.id)} className={styles.logDeleteBtn}>
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                    
                    {log.parsed_card_name && (
                      <div className={styles.logParsed}>
                        <span className={styles.cardName}>{log.parsed_card_name}</span>
                        <span className={styles.amount}>{formatAmount(log.parsed_amount)}</span>
                        {log.parsed_description && (
                          <span className={styles.description}>{log.parsed_description}</span>
                        )}
                      </div>
                    )}
                    
                    {log.error_message && (
                      <div className={styles.error}>{log.error_message}</div>
                    )}
                    
                    <details className={styles.rawText}>
                      <summary>원본 메시지</summary>
                      <pre>{log.raw_text}</pre>
                    </details>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
