<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useDialog, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import ApiService from "@/service/api";
import userStore from "@/stores/model/user";

const router = useRouter();
const message = useMessage();
const dialog = useDialog();
const { locale, t } = useI18n();
const store = userStore();

const loading = ref(true);
const refreshing = ref(false);
const overview = ref(null);
const lastOperation = ref(null);
const broadcastMessage = ref("");
const shutdownSeconds = ref(60);
const shutdownMessage = ref("Server will shutdown in 60 seconds.");
const actionLoading = ref({
  broadcast: false,
  shutdown: false,
  syncRest: false,
  syncSav: false,
  backup: false,
});

const isLogin = computed(() => store.getLoginInfo().isLogin);

const texts = computed(() => {
  if (locale.value === "en") {
    return {
      title: "Ops Overview",
      subtitle: "P4 core operations dashboard for server summary, manual actions, and task status.",
      back: "Back to Home",
      playerWorkspace: "Player Workspace",
      palDefenderWorkspace: "PalDefender Workspace",
      authNoticeTitle: "Action login required",
      authNotice: "You can view the summary without logging in. Manual broadcast, shutdown, sync, and backup require admin auth.",
      summaryTitle: "Server Summary",
      latestBackupTitle: "Latest Backup",
      capabilitiesTitle: "Capabilities",
      dependenciesTitle: "Dependencies",
      tasksTitle: "Task Status",
      actionsTitle: "Manual Actions",
      lastOperationTitle: "Last Operation",
      gameVersion: "Game Version",
      serverName: "Server Name",
      onlinePlayers: "Online Players",
      serverFps: "Server FPS",
      uptime: "Uptime",
      gameDays: "Game Days",
      panelVersion: "Panel Version",
      notAvailable: "Unavailable",
      never: "Never",
      notChecked: "Not Checked",
      configured: "Configured",
      enabled: "Enabled",
      disabled: "Disabled",
      reachable: "Reachable",
      unreachable: "Unreachable",
      fileMissing: "File Missing",
      sourceUnchecked: "Remote/Unchecked",
      operationSuccess: "Operation completed",
      operationFail: "Operation failed",
      broadcastPlaceholder: "Broadcast message to all online players",
      shutdownPlaceholder: "Shutdown notice message",
      playerSync: "Player Sync",
      saveSync: "Save Sync",
      manualBackup: "Manual Backup",
      latestBackupMissing: "Backup record exists but file is missing",
      latestBackupEmpty: "No backup record yet",
      taskLastError: "Last Error",
      taskLastSuccess: "Last Success",
      taskLastFinished: "Last Finished",
      taskSuccessCount: "Success Count",
      taskFailureCount: "Failure Count",
      taskDuration: "Last Duration",
      taskInterval: "Interval",
      taskRunning: "Running",
      taskIdle: "Idle",
      dependencyRest: "REST API",
      dependencyRcon: "RCON",
      dependencyPalDefender: "PalDefender",
      dependencySaveSource: "Save Source",
      saveMode: "Save Mode",
      backupCreated: "Backup created",
    };
  }
  if (locale.value === "ja") {
    return {
      title: "運用ダッシュボード",
      subtitle: "P4 のコア運用画面。サーバー概要、手動操作、タスク状態をまとめて確認できます。",
      back: "ホームへ戻る",
      playerWorkspace: "プレイヤーワークスペース",
      palDefenderWorkspace: "PalDefender ワークスペース",
      authNoticeTitle: "操作にはログインが必要です",
      authNotice: "概要の閲覧は可能ですが、手動ブロードキャスト・シャットダウン・同期・バックアップには管理者ログインが必要です。",
      summaryTitle: "サーバー概要",
      latestBackupTitle: "最新バックアップ",
      capabilitiesTitle: "有効機能",
      dependenciesTitle: "依存状態",
      tasksTitle: "タスク状態",
      actionsTitle: "手動操作",
      lastOperationTitle: "直近の操作結果",
      gameVersion: "ゲームバージョン",
      serverName: "サーバー名",
      onlinePlayers: "オンライン人数",
      serverFps: "サーバー FPS",
      uptime: "稼働時間",
      gameDays: "ゲーム日数",
      panelVersion: "パネルバージョン",
      notAvailable: "未取得",
      never: "なし",
      notChecked: "未検査",
      configured: "設定済み",
      enabled: "有効",
      disabled: "無効",
      reachable: "到達可能",
      unreachable: "到達不可",
      fileMissing: "ファイルなし",
      sourceUnchecked: "リモート/未検査",
      operationSuccess: "操作が完了しました",
      operationFail: "操作に失敗しました",
      broadcastPlaceholder: "オンライン中の全プレイヤーに送るメッセージ",
      shutdownPlaceholder: "シャットダウン通知メッセージ",
      playerSync: "プレイヤー同期",
      saveSync: "セーブ同期",
      manualBackup: "手動バックアップ",
      latestBackupMissing: "バックアップ記録はありますがファイルが見つかりません",
      latestBackupEmpty: "バックアップ記録はまだありません",
      taskLastError: "直近エラー",
      taskLastSuccess: "最終成功",
      taskLastFinished: "最終終了",
      taskSuccessCount: "成功回数",
      taskFailureCount: "失敗回数",
      taskDuration: "最終実行時間",
      taskInterval: "実行間隔",
      taskRunning: "実行中",
      taskIdle: "待機中",
      dependencyRest: "REST API",
      dependencyRcon: "RCON",
      dependencyPalDefender: "PalDefender",
      dependencySaveSource: "セーブソース",
      saveMode: "セーブモード",
      backupCreated: "バックアップ作成済み",
    };
  }
  return {
    title: "运维总览",
    subtitle: "P4 核心运维面板：统一查看服务总览、手动动作和任务状态。",
    back: "返回首页",
    playerWorkspace: "玩家工作台",
    palDefenderWorkspace: "PalDefender 工作台",
    authNoticeTitle: "操作需要登录",
    authNotice: "未登录时可以查看总览，但手动广播、关服、同步和备份需要管理员登录。",
    summaryTitle: "服务总览",
    latestBackupTitle: "最近备份",
    capabilitiesTitle: "启用能力",
    dependenciesTitle: "依赖状态",
    tasksTitle: "任务状态",
    actionsTitle: "手动动作",
    lastOperationTitle: "最近操作结果",
    gameVersion: "游戏版本",
    serverName: "服务器名称",
    onlinePlayers: "在线人数",
    serverFps: "服务 FPS",
    uptime: "运行时长",
    gameDays: "游戏天数",
    panelVersion: "面板版本",
    notAvailable: "未获取",
    never: "从未",
    notChecked: "未检测",
    configured: "已配置",
    enabled: "已启用",
    disabled: "未启用",
    reachable: "可达",
    unreachable: "不可达",
    fileMissing: "文件缺失",
    sourceUnchecked: "远端/未检测",
    operationSuccess: "操作完成",
    operationFail: "操作失败",
    broadcastPlaceholder: "向所有在线玩家广播的消息",
    shutdownPlaceholder: "关服通知消息",
    playerSync: "玩家同步",
    saveSync: "存档同步",
    manualBackup: "手动备份",
    latestBackupMissing: "备份记录存在，但文件已经缺失",
    latestBackupEmpty: "还没有备份记录",
    taskLastError: "最近错误",
    taskLastSuccess: "最近成功",
    taskLastFinished: "最近结束",
    taskSuccessCount: "成功次数",
    taskFailureCount: "失败次数",
    taskDuration: "最近耗时",
    taskInterval: "执行间隔",
    taskRunning: "运行中",
    taskIdle: "空闲",
    dependencyRest: "REST API",
    dependencyRcon: "RCON",
    dependencyPalDefender: "PalDefender",
    dependencySaveSource: "存档来源",
    saveMode: "存档模式",
    backupCreated: "备份已创建",
  };
});

const taskLabels = computed(() => ({
  player_sync: texts.value.playerSync,
  save_sync: texts.value.saveSync,
  backup: t("button.backup"),
  cache_cleanup: "Cache Cleanup",
}));

const capabilityItems = computed(() => {
  if (!overview.value) return [];
  const capabilities = overview.value.capabilities || {};
  return [
    { label: "REST", enabled: !!capabilities.rest_enabled },
    { label: "RCON", enabled: !!capabilities.rcon_configured },
    { label: "PalDefender", enabled: !!capabilities.paldefender_enabled },
    { label: texts.value.playerSync, enabled: !!capabilities.player_sync_enabled },
    { label: texts.value.saveSync, enabled: !!capabilities.save_sync_enabled },
    { label: t("button.backup"), enabled: !!capabilities.backup_enabled },
    { label: "Kick Non-Whitelist", enabled: !!capabilities.kick_non_whitelist },
    { label: "Player Logging", enabled: !!capabilities.player_logging },
  ];
});

const dependencyItems = computed(() => {
  if (!overview.value) return [];
  const dependencies = overview.value.dependencies || {};
  return [
    { key: "rest", label: texts.value.dependencyRest, value: dependencies.rest || {} },
    { key: "rcon", label: texts.value.dependencyRcon, value: dependencies.rcon || {} },
    { key: "paldefender", label: texts.value.dependencyPalDefender, value: dependencies.paldefender || {} },
    { key: "save_source", label: texts.value.dependencySaveSource, value: dependencies.save_source || {} },
  ];
});

const taskItems = computed(() => {
  if (!overview.value?.tasks) return [];
  const tasks = overview.value.tasks;
  return [tasks.player_sync, tasks.save_sync, tasks.backup, tasks.cache_cleanup].filter(Boolean);
});

const latestBackup = computed(() => overview.value?.latest_backup || null);

const formatDateTime = (value) => {
  if (!value) return texts.value.never;
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format("YYYY-MM-DD HH:mm:ss") : String(value);
};

const formatDuration = (ms) => {
  if (!ms && ms !== 0) return texts.value.notAvailable;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
};

const formatUptime = (seconds) => {
  if (!seconds && seconds !== 0) return texts.value.notAvailable;
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return days > 0 ? `${days}d ${hours}h ${minutes}m` : `${hours}h ${minutes}m`;
};

const formatBytes = (size) => {
  if (!size && size !== 0) return texts.value.notAvailable;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`;
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`;
};

const dependencyTagType = (dependency) => {
  if (!dependency?.enabled) return "default";
  if (dependency.checked && dependency.reachable) return "success";
  if (dependency.checked && !dependency.reachable) return "error";
  return "warning";
};

const dependencyStatusText = (dependency) => {
  if (!dependency?.enabled) return texts.value.disabled;
  if (!dependency.checked) {
    if (dependency.mode) {
      return `${texts.value.sourceUnchecked} · ${texts.value.saveMode}: ${dependency.mode}`;
    }
    return texts.value.notChecked;
  }
  return dependency.reachable ? texts.value.reachable : texts.value.unreachable;
};

const ensureAuth = () => {
  if (!isLogin.value) {
    message.warning(texts.value.authNotice);
    return false;
  }
  return true;
};

const applyOperationResult = (payload) => {
  lastOperation.value = {
    ...payload,
    recorded_at: new Date().toISOString(),
  };
};

const fetchOverview = async (silent = false) => {
  if (silent) {
    refreshing.value = true;
  } else {
    loading.value = true;
  }
  const { data, statusCode } = await new ApiService().getServerOverview();
  if (statusCode.value === 200 && data.value) {
    overview.value = data.value;
  } else if (!silent) {
    message.error(texts.value.operationFail);
  }
  loading.value = false;
  refreshing.value = false;
};

const goHome = () => {
  router.push({ name: "home" });
};

const handleBroadcast = async () => {
  if (!ensureAuth()) return;
  if (!broadcastMessage.value.trim()) {
    message.warning(texts.value.broadcastPlaceholder);
    return;
  }
  actionLoading.value.broadcast = true;
  const { data, statusCode } = await new ApiService().sendBroadcast({
    message: broadcastMessage.value,
  });
  actionLoading.value.broadcast = false;
  if (statusCode.value === 200) {
    applyOperationResult(data.value || {});
    message.success(t("message.broadcastsuccess"));
    broadcastMessage.value = "";
    fetchOverview(true);
    return;
  }
  message.error(data.value?.error || texts.value.operationFail);
};

const handleShutdown = async () => {
  if (!ensureAuth()) return;
  if (!shutdownMessage.value.trim()) {
    message.warning(texts.value.shutdownPlaceholder);
    return;
  }
  dialog.warning({
    title: t("message.warn"),
    content: t("message.shutdowntip"),
    positiveText: t("button.confirm"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      actionLoading.value.shutdown = true;
      const { data, statusCode } = await new ApiService().shutdownServer({
        seconds: shutdownSeconds.value || 60,
        message: shutdownMessage.value,
      });
      actionLoading.value.shutdown = false;
      if (statusCode.value === 200) {
        applyOperationResult(data.value || {});
        message.success(t("message.shutdownsuccess"));
        fetchOverview(true);
        return;
      }
      message.error(data.value?.error || texts.value.operationFail);
    },
  });
};

const handleSync = async (from) => {
  if (!ensureAuth()) return;
  const key = from === "rest" ? "syncRest" : "syncSav";
  actionLoading.value[key] = true;
  const { data, statusCode } = await new ApiService().syncServer({ from });
  actionLoading.value[key] = false;
  if (statusCode.value === 200) {
    applyOperationResult(data.value || {});
    message.success(texts.value.operationSuccess);
    fetchOverview(true);
    return;
  }
  message.error(data.value?.error || texts.value.operationFail);
};

const handleBackup = async () => {
  if (!ensureAuth()) return;
  actionLoading.value.backup = true;
  const { data, statusCode } = await new ApiService().createBackup();
  actionLoading.value.backup = false;
  if (statusCode.value === 200) {
    applyOperationResult(data.value || {});
    message.success(texts.value.backupCreated);
    fetchOverview(true);
    return;
  }
  message.error(data.value?.error || texts.value.operationFail);
};

let timer = null;
onMounted(async () => {
  await fetchOverview();
  timer = window.setInterval(() => fetchOverview(true), 15000);
});

onUnmounted(() => {
  if (timer) {
    window.clearInterval(timer);
  }
});
</script>

<template>
  <div class="min-h-screen bg-slate-100 dark:bg-slate-950 p-4 md:p-6">
    <div class="mx-auto max-w-7xl">
      <n-space justify="space-between" align="center" class="mb-4" wrap>
        <div>
          <h1 class="text-3xl font-semibold m-0">{{ texts.title }}</h1>
          <p class="mt-2 mb-0 text-slate-500 dark:text-slate-300">
            {{ texts.subtitle }}
          </p>
        </div>
        <n-space>
          <n-button secondary strong @click="router.push('/players')">{{ texts.playerWorkspace }}</n-button>
          <n-button secondary strong @click="router.push('/paldefender')">{{ texts.palDefenderWorkspace }}</n-button>
          <n-button secondary strong @click="goHome">{{ texts.back }}</n-button>
          <n-button type="primary" secondary strong :loading="refreshing" @click="fetchOverview(true)">
            {{ t("button.refreshStatus") }}
          </n-button>
        </n-space>
      </n-space>

      <n-alert
        v-if="!isLogin"
        type="warning"
        class="mb-4"
        :title="texts.authNoticeTitle"
      >
        {{ texts.authNotice }}
      </n-alert>

      <n-spin :show="loading && !overview">
        <n-grid cols="1 s:2 l:4" responsive="screen" x-gap="12" y-gap="12" class="mb-4">
          <n-gi>
            <n-card>
              <n-statistic :label="texts.serverName" :value="overview?.server?.name || texts.notAvailable" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.gameVersion" :value="overview?.server?.version || texts.notAvailable" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.onlinePlayers" :value="overview?.metrics?.current_player_num ?? 0" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.serverFps" :value="overview?.metrics?.server_fps ?? overview?.metrics?.serverFps ?? 0" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.uptime" :value="formatUptime(overview?.metrics?.uptime)" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.gameDays" :value="overview?.metrics?.days ?? 0" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.panelVersion" :value="overview?.panel_version || texts.notAvailable" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card>
              <n-statistic :label="texts.latestBackupTitle" :value="latestBackup?.path || texts.latestBackupEmpty">
                <template #suffix>
                  <n-tag :type="latestBackup?.file_exists ? 'success' : 'warning'" size="small" round>
                    {{ latestBackup?.file_exists ? texts.reachable : texts.fileMissing }}
                  </n-tag>
                </template>
              </n-statistic>
            </n-card>
          </n-gi>
        </n-grid>

        <n-grid cols="1 l:2" responsive="screen" x-gap="12" y-gap="12" class="mb-4">
          <n-gi>
            <n-card :title="texts.actionsTitle">
              <n-space vertical size="large">
                <div>
                  <div class="mb-2 font-medium">{{ t("button.broadcast") }}</div>
                  <n-space>
                    <n-input v-model:value="broadcastMessage" :placeholder="texts.broadcastPlaceholder" />
                    <n-button type="primary" :loading="actionLoading.broadcast" :disabled="!isLogin" @click="handleBroadcast">
                      {{ t("button.broadcast") }}
                    </n-button>
                  </n-space>
                </div>

                <div>
                  <div class="mb-2 font-medium">{{ t("button.shutdown") }}</div>
                  <n-space vertical>
                    <n-input-number v-model:value="shutdownSeconds" :min="1" :max="3600" />
                    <n-input v-model:value="shutdownMessage" :placeholder="texts.shutdownPlaceholder" />
                    <n-button type="error" :loading="actionLoading.shutdown" :disabled="!isLogin" @click="handleShutdown">
                      {{ t("button.shutdown") }}
                    </n-button>
                  </n-space>
                </div>

                <div>
                  <div class="mb-2 font-medium">{{ texts.playerSync }} / {{ texts.saveSync }}</div>
                  <n-space>
                    <n-button secondary strong type="primary" :loading="actionLoading.syncRest" :disabled="!isLogin" @click="handleSync('rest')">
                      {{ texts.playerSync }}
                    </n-button>
                    <n-button secondary strong type="primary" :loading="actionLoading.syncSav" :disabled="!isLogin" @click="handleSync('sav')">
                      {{ texts.saveSync }}
                    </n-button>
                    <n-button secondary strong type="success" :loading="actionLoading.backup" :disabled="!isLogin" @click="handleBackup">
                      {{ texts.manualBackup }}
                    </n-button>
                  </n-space>
                </div>
              </n-space>
            </n-card>
          </n-gi>

          <n-gi>
            <n-card :title="texts.lastOperationTitle">
              <n-empty v-if="!lastOperation" :description="texts.notAvailable" />
              <n-space v-else vertical>
                <n-tag type="success" round>{{ lastOperation.action || texts.operationSuccess }}</n-tag>
                <div>{{ lastOperation.message || texts.operationSuccess }}</div>
                <div>{{ formatDateTime(lastOperation.recorded_at) }}</div>
                <n-code :code="JSON.stringify(lastOperation, null, 2)" language="json" word-wrap />
              </n-space>
            </n-card>
          </n-gi>
        </n-grid>

        <n-grid cols="1 l:2" responsive="screen" x-gap="12" y-gap="12" class="mb-4">
          <n-gi>
            <n-card :title="texts.capabilitiesTitle">
              <n-space>
                <n-tag v-for="item in capabilityItems" :key="item.label" :type="item.enabled ? 'success' : 'default'" round>
                  {{ item.label }} · {{ item.enabled ? texts.enabled : texts.disabled }}
                </n-tag>
              </n-space>
            </n-card>
          </n-gi>

          <n-gi>
            <n-card :title="texts.dependenciesTitle">
              <n-space vertical>
                <div v-for="item in dependencyItems" :key="item.key" class="flex justify-between items-start gap-3">
                  <div>
                    <div class="font-medium">{{ item.label }}</div>
                    <div class="text-sm text-slate-500 dark:text-slate-300">
                      {{ dependencyStatusText(item.value) }}
                    </div>
                    <div v-if="item.value.error" class="text-sm text-rose-500 mt-1">{{ item.value.error }}</div>
                  </div>
                  <n-tag :type="dependencyTagType(item.value)" round>
                    {{ dependencyStatusText(item.value) }}
                  </n-tag>
                </div>
              </n-space>
            </n-card>
          </n-gi>
        </n-grid>

        <n-card :title="texts.latestBackupTitle" class="mb-4">
          <n-empty v-if="!latestBackup" :description="texts.latestBackupEmpty" />
          <n-space v-else vertical>
            <div>{{ latestBackup.path }}</div>
            <div>{{ formatDateTime(latestBackup.save_time) }}</div>
            <div>{{ formatBytes(latestBackup.size_bytes) }}</div>
            <n-tag :type="latestBackup.file_exists ? 'success' : 'warning'" round>
              {{ latestBackup.file_exists ? texts.reachable : texts.latestBackupMissing }}
            </n-tag>
          </n-space>
        </n-card>

        <n-card :title="texts.tasksTitle">
          <n-grid cols="1 m:2" responsive="screen" x-gap="12" y-gap="12">
            <n-gi v-for="taskItem in taskItems" :key="taskItem.name">
              <n-card size="small">
                <n-space justify="space-between" align="center">
                  <div class="font-medium">{{ taskLabels[taskItem.name] || taskItem.name }}</div>
                  <n-tag :type="taskItem.running ? 'warning' : taskItem.enabled ? 'success' : 'default'" round>
                    {{ taskItem.running ? texts.taskRunning : texts.taskIdle }}
                  </n-tag>
                </n-space>
                <n-space vertical size="small" class="mt-3 text-sm">
                  <div>{{ texts.taskInterval }}：{{ taskItem.interval_seconds || 0 }}s</div>
                  <div>{{ texts.taskDuration }}：{{ formatDuration(taskItem.last_duration_ms) }}</div>
                  <div>{{ texts.taskLastSuccess }}：{{ formatDateTime(taskItem.last_success_at) }}</div>
                  <div>{{ texts.taskLastFinished }}：{{ formatDateTime(taskItem.last_finished_at) }}</div>
                  <div>{{ texts.taskSuccessCount }}：{{ taskItem.success_count || 0 }}</div>
                  <div>{{ texts.taskFailureCount }}：{{ taskItem.failure_count || 0 }}</div>
                  <div>{{ texts.taskLastError }}：{{ taskItem.last_error_code || taskItem.last_error || texts.notAvailable }}</div>
                </n-space>
              </n-card>
            </n-gi>
          </n-grid>
        </n-card>
      </n-spin>
    </div>
  </div>
</template>
