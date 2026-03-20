<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import palMap from "@/assets/pal.json";
import palItems from "@/assets/items.json";
import { explainPalDefenderError, isPlayerOnlineForLiveGrant } from "@/utils/paldefender";

const props = defineProps({
  playerList: {
    type: Array,
    default: () => [],
  },
  guildList: {
    type: Array,
    default: () => [],
  },
  compact: {
    type: Boolean,
    default: false,
  },
});

const { t, locale } = useI18n();
const message = useMessage();

const activeTab = ref("grant");
const status = ref(null);
const loadingStatus = ref(false);
const auditLogs = ref([]);
const loadingAudit = ref(false);
const submitting = ref(false);
const exportingAudit = ref(false);
const retryingBatch = ref(false);
const selectedPresetNames = ref([]);
const targetMode = ref("players");
const selectedPlayerIds = ref([]);
const selectedGuildAdminPlayerUid = ref(null);
const onlineOnly = ref(true);
const latestResult = ref(null);
const auditFilters = reactive({
  action: "",
  batch_id: "",
  player_uid: "",
  success: "all",
  error_code: "",
  limit: 50,
});

const createEmptyGrantPlan = () => ({
  exp: null,
  lifmunks: null,
  technology_points: null,
  ancient_technology_points: null,
  items: [],
  pals: [],
  pal_eggs: [],
  pal_templates: [],
});

const grantPlan = reactive(createEmptyGrantPlan());

const createItemRow = () => ({ item_id: null, amount: 1 });
const createPalRow = () => ({ pal_id: null, level: 1, amount: 1 });
const createEggRow = () => ({ item_id: null, pal_id: null, level: 1, amount: 1 });
const createTemplateRow = () => ({ template_name: "", amount: 1 });

const auditTexts = computed(() => {
  if (locale.value === "en") {
    return {
      actionAll: "All Actions",
      actionBatchGrant: "Batch Grant",
      actionBatchGrantRetry: "Batch Retry",
      successAll: "All Results",
      successOnly: "Success Only",
      failedOnly: "Failed Only",
      exportAudit: "Export Audit",
      retryFailed: "Retry Failed",
      batchIdPlaceholder: "Batch ID",
      playerUidPlaceholder: "Player UID",
      errorCodePlaceholder: "Error Code",
      resetFilters: "Reset Filters",
      latestBatchTitle: "Latest Batch Result",
      sourceBatch: "Source Batch",
      duration: "Duration",
      completedAt: "Completed",
      requestedTargets: "Requested",
      appliedPresets: "Applied Presets",
      failureCodes: "Failure Codes",
      retryHint: "Retry target batch",
      operator: "Operator",
      grantSummary: "Grant Summary",
      exportSuccess: "Audit logs exported",
      exportFail: "Failed to export audit logs",
      retryBatchMissing: "No retryable batch found yet",
      retrySuccess: "Retry submitted for {count} player(s)",
    };
  }
  if (locale.value === "ja") {
    return {
      actionAll: "全アクション",
      actionBatchGrant: "一括付与",
      actionBatchGrantRetry: "再試行",
      successAll: "全結果",
      successOnly: "成功のみ",
      failedOnly: "失敗のみ",
      exportAudit: "監査をエクスポート",
      retryFailed: "失敗分を再試行",
      batchIdPlaceholder: "バッチ ID",
      playerUidPlaceholder: "Player UID",
      errorCodePlaceholder: "エラーコード",
      resetFilters: "フィルターをリセット",
      latestBatchTitle: "最新バッチ結果",
      sourceBatch: "元バッチ",
      duration: "所要時間",
      completedAt: "完了時刻",
      requestedTargets: "要求対象数",
      appliedPresets: "適用プリセット",
      failureCodes: "失敗コード",
      retryHint: "再試行対象バッチ",
      operator: "操作者",
      grantSummary: "付与概要",
      exportSuccess: "監査ログをエクスポートしました",
      exportFail: "監査ログのエクスポートに失敗しました",
      retryBatchMissing: "再試行できるバッチがありません",
      retrySuccess: "{count} 人に対する再試行を送信しました",
    };
  }
  return {
    actionAll: "全部动作",
    actionBatchGrant: "批量发放",
    actionBatchGrantRetry: "失败重试",
    successAll: "全部结果",
    successOnly: "仅成功",
    failedOnly: "仅失败",
    exportAudit: "导出审计",
    retryFailed: "重试失败批次",
    batchIdPlaceholder: "批次 ID",
    playerUidPlaceholder: "玩家 UID",
    errorCodePlaceholder: "错误码",
    resetFilters: "重置筛选",
    latestBatchTitle: "最近批次结果",
    sourceBatch: "源批次",
    duration: "耗时",
    completedAt: "完成时间",
    requestedTargets: "请求目标数",
    appliedPresets: "已应用预设",
    failureCodes: "失败码统计",
    retryHint: "当前可重试批次",
    operator: "操作人",
    grantSummary: "发放摘要",
    exportSuccess: "审计日志已导出",
    exportFail: "导出审计日志失败",
    retryBatchMissing: "当前没有可重试的失败批次",
    retrySuccess: "已向 {count} 名玩家提交重试",
  };
});

const itemCatalog = computed(() => palItems[locale.value] || palItems.zh || []);
const itemOptions = computed(() =>
  itemCatalog.value.map((item) => ({
    label: `${item.name} (${item.key || item.id})`,
    value: item.key || item.id,
  }))
);
const eggOptions = computed(() =>
  itemCatalog.value
    .filter((item) => /^PalEgg_/i.test(item.key || item.id))
    .map((item) => ({
      label: `${item.name} (${item.key || item.id})`,
      value: item.key || item.id,
    }))
);
const palOptions = computed(() =>
  Object.entries(palMap[locale.value] || palMap.zh || {}).map(([key, value]) => ({
    label: `${value} (${key})`,
    value: key,
  }))
);

const auditActionOptions = computed(() => [
  { label: auditTexts.value.actionAll, value: "" },
  { label: auditTexts.value.actionBatchGrant, value: "batch-grant" },
  { label: auditTexts.value.actionBatchGrantRetry, value: "batch-grant-retry" },
]);

const auditSuccessOptions = computed(() => [
  { label: auditTexts.value.successAll, value: "all" },
  { label: auditTexts.value.successOnly, value: "success" },
  { label: auditTexts.value.failedOnly, value: "failed" },
]);

const playerLookup = computed(() => {
  const lookup = new Map();
  (props.playerList || []).forEach((player) => {
    lookup.set(player.player_uid, player);
  });
  return lookup;
});

const playerOptions = computed(() =>
  (props.playerList || []).map((player) => ({
    label: `[${isPlayerOnlineForLiveGrant(player.last_online) ? t("status.online") : t("status.offline")}] ${player.nickname} (${player.player_uid})`,
    value: player.player_uid,
  }))
);

const guildOptions = computed(() =>
  (props.guildList || []).map((guild) => ({
    label: `${guild.name} (${guild.players?.length || 0})`,
    value: guild.admin_player_uid,
  }))
);

const selectedGuild = computed(() =>
  (props.guildList || []).find((guild) => guild.admin_player_uid === selectedGuildAdminPlayerUid.value) || null
);

const selectedPresets = computed(() => {
  const presetLookup = new Map((status.value?.presets || []).map((preset) => [preset.name, preset]));
  return selectedPresetNames.value.map((name) => presetLookup.get(name)).filter(Boolean);
});

const resolvedTargets = computed(() => {
  const targets = [];
  const pushTarget = (source) => {
    if (!source?.player_uid) {
      return;
    }
    const player = playerLookup.value.get(source.player_uid) || source;
    if (onlineOnly.value && !isPlayerOnlineForLiveGrant(player.last_online)) {
      return;
    }
    if (targets.some((target) => target.player_uid === source.player_uid)) {
      return;
    }
    targets.push({
      player_uid: source.player_uid,
      user_id: player.user_id || source.user_id || "",
      steam_id: player.steam_id || source.steam_id || "",
      nickname: player.nickname || source.nickname || source.player_uid,
      last_online: player.last_online || "",
    });
  };
  if (targetMode.value === "online") {
    (props.playerList || []).forEach((player) => {
      if (isPlayerOnlineForLiveGrant(player.last_online)) {
        pushTarget(player);
      }
    });
    return targets;
  }
  if (targetMode.value === "guild") {
    (selectedGuild.value?.players || []).forEach((player) => pushTarget(player));
    return targets;
  }
  selectedPlayerIds.value.forEach((playerUid) => {
    pushTarget(playerLookup.value.get(playerUid));
  });
  return targets;
});

const checkedAtText = computed(() => {
  if (!status.value?.checked_at) {
    return "-";
  }
  return dayjs(status.value.checked_at).format("YYYY-MM-DD HH:mm:ss");
});

const statusTagType = computed(() => {
  if (!status.value) {
    return "default";
  }
  if (status.value.healthy) {
    return "success";
  }
  if (!status.value.enabled) {
    return "default";
  }
  return "warning";
});

const statusLabel = computed(() => {
  if (!status.value) {
    return t("message.loading");
  }
  if (status.value.healthy) {
    return t("message.palDefenderStatusReady");
  }
  if (!status.value.enabled) {
    return t("message.palDefenderStatusDisabled");
  }
  if (!status.value.configured) {
    return t("message.palDefenderStatusUnconfigured");
  }
  return t("message.fail");
});

const normalizeText = (value) => String(value ?? "").trim();

const retryBatchId = computed(() => {
  const filteredBatchID = normalizeText(auditFilters.batch_id);
  if (filteredBatchID) {
    return filteredBatchID;
  }
  const latestBatchID = normalizeText(latestResult.value?.batch_id);
  if (latestResult.value?.failure_count > 0 && latestBatchID) {
    return latestBatchID;
  }
  const firstFailedLog = auditLogs.value.find((log) => !log.success && normalizeText(log.batch_id));
  return normalizeText(firstFailedLog?.batch_id);
});

const getPositiveNumber = (value, allowZero = false) => {
  const parsed = Number.parseInt(String(value ?? 0), 10);
  if (!Number.isFinite(parsed)) {
    return 0;
  }
  if (allowZero && parsed === 0) {
    return 0;
  }
  return parsed > 0 ? parsed : 0;
};

const resetGrantPlan = () => {
  Object.assign(grantPlan, createEmptyGrantPlan());
  selectedPresetNames.value = [];
  latestResult.value = null;
};

const resetAuditFilters = () => {
  auditFilters.action = "";
  auditFilters.batch_id = "";
  auditFilters.player_uid = "";
  auditFilters.success = "all";
  auditFilters.error_code = "";
  auditFilters.limit = 50;
};

const buildGrantPayload = () => ({
  exp: getPositiveNumber(grantPlan.exp, true),
  lifmunks: getPositiveNumber(grantPlan.lifmunks, true),
  technology_points: getPositiveNumber(grantPlan.technology_points, true),
  ancient_technology_points: getPositiveNumber(grantPlan.ancient_technology_points, true),
  items: grantPlan.items
    .filter((item) => item.item_id && getPositiveNumber(item.amount) > 0)
    .map((item) => ({ item_id: item.item_id, amount: getPositiveNumber(item.amount) })),
  pals: grantPlan.pals
    .filter((pal) => pal.pal_id && getPositiveNumber(pal.amount) > 0)
    .map((pal) => ({
      pal_id: pal.pal_id,
      level: getPositiveNumber(pal.level) || 1,
      amount: getPositiveNumber(pal.amount),
    })),
  pal_eggs: grantPlan.pal_eggs
    .filter((egg) => egg.item_id && egg.pal_id && getPositiveNumber(egg.amount) > 0)
    .map((egg) => ({
      item_id: egg.item_id,
      pal_id: egg.pal_id,
      level: getPositiveNumber(egg.level) || 1,
      amount: getPositiveNumber(egg.amount),
    })),
  pal_templates: grantPlan.pal_templates
    .filter((template) => template.template_name?.trim() && getPositiveNumber(template.amount) > 0)
    .map((template) => ({
      template_name: template.template_name.trim(),
      amount: getPositiveNumber(template.amount),
    })),
});

const buildAuditQuery = () => {
  const query = { limit: auditFilters.limit };
  const action = normalizeText(auditFilters.action);
  const batchID = normalizeText(auditFilters.batch_id);
  const playerUID = normalizeText(auditFilters.player_uid);
  const errorCode = normalizeText(auditFilters.error_code);
  if (action) {
    query.action = action;
  }
  if (batchID) {
    query.batch_id = batchID;
  }
  if (playerUID) {
    query.player_uid = playerUID;
  }
  if (auditFilters.success === "success") {
    query.success = true;
  } else if (auditFilters.success === "failed") {
    query.success = false;
  }
  if (errorCode) {
    query.error_code = errorCode;
  }
  return query;
};

const loadStatus = async () => {
  loadingStatus.value = true;
  const { data, statusCode } = await new ApiService().getPalDefenderStatus();
  loadingStatus.value = false;
  if (statusCode.value === 200) {
    status.value = data.value;
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
};

const loadAuditLogs = async () => {
  loadingAudit.value = true;
  const { data, statusCode } = await new ApiService().getPalDefenderAuditLogs(buildAuditQuery());
  loadingAudit.value = false;
  if (statusCode.value === 200) {
    auditLogs.value = data.value || [];
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
};

const applyBatchResult = async (payload, mode) => {
  latestResult.value = payload;
  activeTab.value = mode === "retry" ? "audit" : "grant";
  if (payload?.failure_count > 0 && payload?.success_count > 0) {
    message.warning(
      t("message.batchGrantPartial", {
        success: payload.success_count,
        fail: payload.failure_count,
      })
    );
  } else if (payload?.failure_count > 0) {
    message.error(
      t("message.batchGrantFail", {
        err: payload?.results?.find((item) => item.error)?.error || t("message.fail"),
      })
    );
  } else if (mode === "retry") {
    message.success(
      auditTexts.value.retrySuccess.replace("{count}", String(payload?.success_count || 0))
    );
  } else {
    message.success(t("message.batchGrantSuccess", { count: payload?.success_count || 0 }));
  }
  await loadAuditLogs();
};

const submitBatchGrant = async () => {
  if (!resolvedTargets.value.length) {
    message.warning(t("message.batchTargetRequired"));
    return;
  }
  const grant = buildGrantPayload();
  if (
    !selectedPresetNames.value.length &&
    !grant.items.length &&
    !grant.pals.length &&
    !grant.pal_eggs.length &&
    !grant.pal_templates.length &&
    !grant.exp &&
    !grant.lifmunks &&
    !grant.technology_points &&
    !grant.ancient_technology_points
  ) {
    message.warning(t("message.noGrantPlanConfigured"));
    return;
  }
  submitting.value = true;
  const { data, statusCode } = await new ApiService().grantPalDefenderBatch({
    targets: resolvedTargets.value.map((target) => ({
      player_uid: target.player_uid,
      user_id: target.user_id,
      steam_id: target.steam_id,
    })),
    preset_names: selectedPresetNames.value,
    grant,
  });
  submitting.value = false;
  if (statusCode.value === 200) {
    await applyBatchResult(data.value, "grant");
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
};

const exportAuditLogs = async () => {
  exportingAudit.value = true;
  try {
    const request = await new ApiService().exportPalDefenderAuditLogs(buildAuditQuery());
    await request.execute();
    if (request.statusCode.value === 200 && request.data.value) {
      const url = URL.createObjectURL(request.data.value);
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute(
        "download",
        `paldefender-audit-${dayjs().format("YYYYMMDD-HHmmss")}.json`
      );
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      message.success(auditTexts.value.exportSuccess);
      return;
    }
    message.error(auditTexts.value.exportFail);
  } catch (error) {
    console.error("PalDefender audit export failed", error);
    message.error(auditTexts.value.exportFail);
  } finally {
    exportingAudit.value = false;
  }
};

const retryFailedBatch = async () => {
  if (!retryBatchId.value) {
    message.warning(auditTexts.value.retryBatchMissing);
    return;
  }
  retryingBatch.value = true;
  const { data, statusCode } = await new ApiService().retryPalDefenderBatch({
    batch_id: retryBatchId.value,
    failed_only: true,
  });
  retryingBatch.value = false;
  if (statusCode.value === 200) {
    await applyBatchResult(data.value, "retry");
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
};

const formatFailureCodes = (failureCodes) => {
  if (!failureCodes || typeof failureCodes !== "object") {
    return "";
  }
  return Object.entries(failureCodes)
    .map(([code, count]) => `${code || "unknown"} × ${count}`)
    .join(", ");
};

const formatGrantSummary = (grant) => {
  if (!grant || typeof grant !== "object") {
    return "";
  }
  const summary = [];
  if (Number(grant.exp || 0) > 0) {
    summary.push(`EXP ${grant.exp}`);
  }
  if (Number(grant.lifmunks || 0) > 0) {
    summary.push(`Lifmunks ${grant.lifmunks}`);
  }
  if (Number(grant.technology_points || 0) > 0) {
    summary.push(`Tech ${grant.technology_points}`);
  }
  if (Number(grant.ancient_technology_points || 0) > 0) {
    summary.push(`Ancient ${grant.ancient_technology_points}`);
  }
  if (Array.isArray(grant.items) && grant.items.length) {
    summary.push(`Items ${grant.items.length}`);
  }
  if (Array.isArray(grant.pals) && grant.pals.length) {
    summary.push(`Pals ${grant.pals.length}`);
  }
  if (Array.isArray(grant.pal_eggs) && grant.pal_eggs.length) {
    summary.push(`Eggs ${grant.pal_eggs.length}`);
  }
  if (Array.isArray(grant.pal_templates) && grant.pal_templates.length) {
    summary.push(`Templates ${grant.pal_templates.length}`);
  }
  return summary.join(" · ");
};

const formatAuditError = (log) => explainPalDefenderError(t, log, log?.error || t("message.success"));

onMounted(async () => {
  await Promise.all([loadStatus(), loadAuditLogs()]);
});
</script>

<template>
  <n-card size="small" :bordered="false" class="mt-4">
    <template #header>
      {{ $t("button.batchGrant") }}
    </template>
    <template #header-extra>
      <n-space wrap>
        <n-button size="small" tertiary :loading="loadingStatus" @click="loadStatus">
          {{ $t("button.refreshStatus") }}
        </n-button>
        <n-button size="small" tertiary :loading="loadingAudit" @click="loadAuditLogs">
          {{ $t("button.refreshAudit") }}
        </n-button>
        <n-button size="small" tertiary :loading="exportingAudit" @click="exportAuditLogs">
          {{ auditTexts.exportAudit }}
        </n-button>
        <n-button
          size="small"
          type="warning"
          tertiary
          :loading="retryingBatch"
          :disabled="!retryBatchId"
          @click="retryFailedBatch"
        >
          {{ auditTexts.retryFailed }}
        </n-button>
      </n-space>
    </template>

    <n-space vertical size="small">
      <n-alert :type="statusTagType" :show-icon="false">
        <div class="flex items-center justify-between gap-3 flex-wrap">
          <n-space size="small" align="center">
            <span>PalDefender</span>
            <n-tag :type="statusTagType" round>
              {{ statusLabel }}
            </n-tag>
          </n-space>
          <span class="text-xs opacity-75">{{ $t("item.checkedAt") }}: {{ checkedAtText }}</span>
        </div>
        <div class="mt-2 text-sm">
          {{ status?.error ? explainPalDefenderError(t, status, status.error) : $t("message.palDefenderStatusReady") }}
        </div>
        <div class="mt-2 text-xs opacity-80">
          Address: {{ status?.address || "-" }} | Timeout: {{ status?.timeout || 0 }}s
        </div>
        <div class="mt-2 flex gap-2 flex-wrap">
          <n-tag v-for="capability in status?.capabilities || []" :key="capability" size="small" round>
            {{ capability }}
          </n-tag>
        </div>
      </n-alert>

      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="grant" :tab="$t('button.batchGrant')">
          <n-space vertical size="small">
            <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
              <n-gi>
                <n-space vertical>
                  <n-radio-group v-model:value="targetMode">
                    <n-space>
                      <n-radio-button value="players">{{ $t("button.players") }}</n-radio-button>
                      <n-radio-button value="guild">{{ $t("button.guilds") }}</n-radio-button>
                      <n-radio-button value="online">{{ $t("status.online") }}</n-radio-button>
                    </n-space>
                  </n-radio-group>
                  <n-switch v-model:value="onlineOnly">
                    <template #checked>{{ $t("status.online") }}</template>
                    <template #unchecked>{{ $t("status.offline") }}</template>
                  </n-switch>
                  <n-select
                    v-if="targetMode === 'players'"
                    v-model:value="selectedPlayerIds"
                    multiple
                    filterable
                    :placeholder="$t('input.selectPlayers')"
                    :options="playerOptions"
                  />
                  <n-select
                    v-else-if="targetMode === 'guild'"
                    v-model:value="selectedGuildAdminPlayerUid"
                    filterable
                    :placeholder="$t('input.selectGuild')"
                    :options="guildOptions"
                  />
                  <n-text depth="3">
                    {{ $t("item.targetCount") }}: {{ resolvedTargets.length }}
                  </n-text>
                </n-space>
              </n-gi>
              <n-gi>
                <n-space vertical>
                  <n-select
                    v-model:value="selectedPresetNames"
                    multiple
                    filterable
                    clearable
                    :placeholder="$t('input.selectPreset')"
                    :options="(status?.presets || []).map((preset) => ({ label: preset.name, value: preset.name }))"
                  />
                  <n-space v-if="selectedPresets.length" vertical size="small">
                    <n-tag v-for="preset in selectedPresets" :key="preset.name" type="info" round>
                      {{ preset.name }}
                    </n-tag>
                    <n-text v-for="preset in selectedPresets" :key="`${preset.name}-desc`" depth="3">
                      {{ preset.description || preset.name }}
                    </n-text>
                  </n-space>
                </n-space>
              </n-gi>
            </n-grid>

            <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
              <n-gi>
                <n-space vertical>
                  <div class="text-sm font-semibold">EXP / Support</div>
                  <div class="grid grid-cols-2 gap-2">
                    <n-input-number v-model:value="grantPlan.exp" :min="0" :precision="0" :placeholder="$t('button.quickGiveExp')" />
                    <n-input-number v-model:value="grantPlan.lifmunks" :min="0" :precision="0" :placeholder="$t('button.quickGiveRelic')" />
                    <n-input-number v-model:value="grantPlan.technology_points" :min="0" :precision="0" :placeholder="$t('button.quickGiveTechPoints')" />
                    <n-input-number v-model:value="grantPlan.ancient_technology_points" :min="0" :precision="0" :placeholder="$t('button.quickGiveBossTechPoints')" />
                  </div>
                </n-space>
              </n-gi>
              <n-gi>
                <n-space vertical>
                  <div class="text-sm font-semibold">{{ $t("button.quickGiveItem") }}</div>
                  <n-dynamic-input v-model:value="grantPlan.items" :on-create="createItemRow">
                    <template #default="{ value }">
                      <div class="flex gap-2 w-full">
                        <n-select v-model:value="value.item_id" filterable :options="itemOptions" :placeholder="$t('input.selectItem')" class="flex-1" />
                        <n-input-number v-model:value="value.amount" :min="1" :precision="0" style="width: 110px" :placeholder="$t('input.amount')" />
                      </div>
                    </template>
                  </n-dynamic-input>
                </n-space>
              </n-gi>
            </n-grid>

            <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
              <n-gi>
                <n-space vertical>
                  <div class="text-sm font-semibold">{{ $t("button.quickGivePal") }}</div>
                  <n-dynamic-input v-model:value="grantPlan.pals" :on-create="createPalRow">
                    <template #default="{ value }">
                      <div class="grid grid-cols-3 gap-2 w-full">
                        <n-select v-model:value="value.pal_id" filterable :options="palOptions" :placeholder="$t('input.selectPal')" class="col-span-3 md:col-span-1" />
                        <n-input-number v-model:value="value.level" :min="1" :precision="0" :placeholder="$t('pal.level')" />
                        <n-input-number v-model:value="value.amount" :min="1" :precision="0" :placeholder="$t('input.amount')" />
                      </div>
                    </template>
                  </n-dynamic-input>
                </n-space>
              </n-gi>
              <n-gi>
                <n-space vertical>
                  <div class="text-sm font-semibold">{{ $t("button.quickGiveEgg") }}</div>
                  <n-dynamic-input v-model:value="grantPlan.pal_eggs" :on-create="createEggRow">
                    <template #default="{ value }">
                      <div class="grid grid-cols-4 gap-2 w-full">
                        <n-select v-model:value="value.item_id" filterable :options="eggOptions" :placeholder="$t('input.selectEgg')" class="col-span-4 md:col-span-2" />
                        <n-select v-model:value="value.pal_id" filterable :options="palOptions" :placeholder="$t('input.selectPal')" class="col-span-4 md:col-span-2" />
                        <n-input-number v-model:value="value.level" :min="1" :precision="0" :placeholder="$t('pal.level')" />
                        <n-input-number v-model:value="value.amount" :min="1" :precision="0" :placeholder="$t('input.amount')" />
                      </div>
                    </template>
                  </n-dynamic-input>
                </n-space>
              </n-gi>
            </n-grid>

            <n-space vertical>
              <div class="text-sm font-semibold">{{ $t("button.grantTemplate") }}</div>
              <n-dynamic-input v-model:value="grantPlan.pal_templates" :on-create="createTemplateRow">
                <template #default="{ value }">
                  <div class="flex gap-2 w-full">
                    <n-input v-model:value="value.template_name" :placeholder="$t('input.templateName')" class="flex-1" />
                    <n-input-number v-model:value="value.amount" :min="1" :precision="0" style="width: 110px" :placeholder="$t('input.amount')" />
                  </div>
                </template>
              </n-dynamic-input>
            </n-space>

            <n-space justify="end">
              <n-button tertiary @click="resetGrantPlan">{{ $t("button.clearPlan") }}</n-button>
              <n-button type="primary" strong secondary :loading="submitting" @click="submitBatchGrant">
                {{ $t("button.batchGrant") }}
              </n-button>
            </n-space>

            <n-card v-if="latestResult" size="small" :title="auditTexts.latestBatchTitle">
              <n-space vertical size="small">
                <div class="flex items-center justify-between gap-2 flex-wrap">
                  <n-space size="small" align="center">
                    <n-tag :type="latestResult.success ? 'success' : 'warning'" round>
                      {{ latestResult.success ? $t("message.success") : $t("message.fail") }}
                    </n-tag>
                    <n-tag size="small" type="info" round>{{ latestResult.batch_id }}</n-tag>
                    <n-tag v-if="latestResult.source_batch_id" size="small" round>
                      {{ auditTexts.sourceBatch }}: {{ latestResult.source_batch_id }}
                    </n-tag>
                  </n-space>
                  <span class="text-xs opacity-75">
                    {{ auditTexts.completedAt }}: {{ dayjs(latestResult.completed_at).format("YYYY-MM-DD HH:mm:ss") }}
                  </span>
                </div>
                <div class="grid grid-cols-2 md:grid-cols-5 gap-2 text-sm">
                  <div>{{ auditTexts.requestedTargets }}: {{ latestResult.requested_target_count }}</div>
                  <div>{{ $t("item.targetCount") }}: {{ latestResult.target_count }}</div>
                  <div>{{ $t("item.successCount") }}: {{ latestResult.success_count }}</div>
                  <div>{{ $t("item.failureCount") }}: {{ latestResult.failure_count }}</div>
                  <div>{{ auditTexts.duration }}: {{ latestResult.duration_ms || 0 }} ms</div>
                </div>
                <div v-if="latestResult.applied_preset_names?.length" class="text-xs opacity-75">
                  {{ auditTexts.appliedPresets }}: {{ latestResult.applied_preset_names.join(", ") }}
                </div>
                <div v-if="formatFailureCodes(latestResult.failure_codes)" class="text-xs text-#d03050">
                  {{ auditTexts.failureCodes }}: {{ formatFailureCodes(latestResult.failure_codes) }}
                </div>
              </n-space>
            </n-card>
          </n-space>
        </n-tab-pane>

        <n-tab-pane name="audit" :tab="$t('item.auditLog')">
          <n-space vertical size="small">
            <n-grid :cols="compact ? 1 : 5" :x-gap="12" :y-gap="12">
              <n-gi>
                <n-select v-model:value="auditFilters.action" :options="auditActionOptions" :placeholder="auditTexts.actionAll" />
              </n-gi>
              <n-gi>
                <n-input v-model:value="auditFilters.batch_id" :placeholder="auditTexts.batchIdPlaceholder" @keyup.enter="loadAuditLogs" />
              </n-gi>
              <n-gi>
                <n-input v-model:value="auditFilters.player_uid" :placeholder="auditTexts.playerUidPlaceholder" @keyup.enter="loadAuditLogs" />
              </n-gi>
              <n-gi>
                <n-select v-model:value="auditFilters.success" :options="auditSuccessOptions" />
              </n-gi>
              <n-gi>
                <n-input v-model:value="auditFilters.error_code" :placeholder="auditTexts.errorCodePlaceholder" @keyup.enter="loadAuditLogs" />
              </n-gi>
            </n-grid>

            <div class="flex items-center justify-between gap-3 flex-wrap">
              <n-text depth="3">{{ auditTexts.retryHint }}: {{ retryBatchId || "-" }}</n-text>
              <n-space>
                <n-button tertiary @click="resetAuditFilters">{{ auditTexts.resetFilters }}</n-button>
                <n-button tertiary :loading="loadingAudit" @click="loadAuditLogs">{{ $t("button.search") }}</n-button>
                <n-button tertiary :loading="exportingAudit" @click="exportAuditLogs">{{ auditTexts.exportAudit }}</n-button>
                <n-button type="warning" tertiary :loading="retryingBatch" :disabled="!retryBatchId" @click="retryFailedBatch">
                  {{ auditTexts.retryFailed }}
                </n-button>
              </n-space>
            </div>

            <n-empty v-if="!loadingAudit && auditLogs.length === 0" />
            <n-spin v-else-if="loadingAudit" size="small" />
            <n-list v-else hoverable>
              <n-list-item v-for="log in auditLogs" :key="log.id">
                <div class="flex items-center justify-between gap-3 flex-wrap">
                  <n-space size="small" align="center">
                    <n-tag :type="log.success ? 'success' : 'error'" round>
                      {{ log.success ? $t("message.success") : $t("message.fail") }}
                    </n-tag>
                    <n-tag size="small" round>{{ log.action }}</n-tag>
                    <n-tag v-if="log.batch_id" size="small" round type="info">{{ log.batch_id }}</n-tag>
                  </n-space>
                  <span class="text-xs opacity-75">{{ dayjs(log.created_at).format("YYYY-MM-DD HH:mm:ss") }}</span>
                </div>
                <div class="mt-2 text-sm">
                  {{ log.nickname || log.player_uid || log.user_id || "-" }}
                </div>
                <div class="mt-1 text-xs opacity-75">
                  UID: {{ log.player_uid || "-" }} | UserID: {{ log.user_id || "-" }} | {{ auditTexts.operator }}: {{ log.operator || "-" }}
                </div>
                <div v-if="log.preset_names?.length" class="mt-1 text-xs opacity-75">
                  {{ auditTexts.appliedPresets }}: {{ log.preset_names.join(", ") }}
                </div>
                <div v-if="formatGrantSummary(log.grant)" class="mt-1 text-xs opacity-75">
                  {{ auditTexts.grantSummary }}: {{ formatGrantSummary(log.grant) }}
                </div>
                <div v-if="log.error_code" class="mt-1 text-xs opacity-75">
                  {{ auditTexts.failureCodes }}: {{ log.error_code }}
                </div>
                <div v-if="log.error" class="mt-2 text-xs text-#d03050">
                  {{ formatAuditError(log) }}
                </div>
              </n-list-item>
            </n-list>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-space>
  </n-card>
</template>
