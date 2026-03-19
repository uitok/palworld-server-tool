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

const status = ref(null);
const loadingStatus = ref(false);
const auditLogs = ref([]);
const loadingAudit = ref(false);
const submitting = ref(false);
const selectedPresetNames = ref([]);
const targetMode = ref("players");
const selectedPlayerIds = ref([]);
const selectedGuildAdminPlayerUid = ref(null);
const onlineOnly = ref(true);
const latestResult = ref(null);

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
  const { data, statusCode } = await new ApiService().getPalDefenderAuditLogs({ limit: 20 });
  loadingAudit.value = false;
  if (statusCode.value === 200) {
    auditLogs.value = data.value || [];
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
};

const submitBatchGrant = async () => {
  if (!resolvedTargets.value.length) {
    message.warning(t("message.batchTargetRequired"));
    return;
  }
  const grant = buildGrantPayload();
  if (!selectedPresetNames.value.length && !grant.items.length && !grant.pals.length && !grant.pal_eggs.length && !grant.pal_templates.length && !grant.exp && !grant.lifmunks && !grant.technology_points && !grant.ancient_technology_points) {
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
    latestResult.value = data.value;
    if (data.value?.failure_count > 0 && data.value?.success_count > 0) {
      message.warning(t("message.batchGrantPartial", {
        success: data.value.success_count,
        fail: data.value.failure_count,
      }));
    } else if (data.value?.failure_count > 0) {
      message.error(t("message.batchGrantFail", { err: data.value?.results?.[0]?.error || t("message.fail") }));
    } else {
      message.success(t("message.batchGrantSuccess", { count: data.value?.success_count || 0 }));
    }
    await loadAuditLogs();
    return;
  }
  message.error(explainPalDefenderError(t, data.value, t("message.fail")));
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
      <n-space>
        <n-button size="small" tertiary :loading="loadingStatus" @click="loadStatus">
          {{ $t("button.refreshStatus") }}
        </n-button>
        <n-button size="small" tertiary :loading="loadingAudit" @click="loadAuditLogs">
          {{ $t("button.refreshAudit") }}
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

      <n-tabs type="line" animated>
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

            <n-alert v-if="latestResult" :type="latestResult.success ? 'success' : 'warning'" :show-icon="false">
              <div class="flex gap-4 flex-wrap">
                <span>{{ $t("item.targetCount") }}: {{ latestResult.target_count }}</span>
                <span>{{ $t("item.successCount") }}: {{ latestResult.success_count }}</span>
                <span>{{ $t("item.failureCount") }}: {{ latestResult.failure_count }}</span>
              </div>
            </n-alert>
          </n-space>
        </n-tab-pane>

        <n-tab-pane name="audit" :tab="$t('item.auditLog')">
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
                UID: {{ log.player_uid || "-" }} | UserID: {{ log.user_id || "-" }}
              </div>
              <div v-if="log.preset_names?.length" class="mt-1 text-xs opacity-75">
                Presets: {{ log.preset_names.join(", ") }}
              </div>
              <div v-if="log.error" class="mt-2 text-xs text-#d03050">
                {{ formatAuditError(log) }}
              </div>
            </n-list-item>
          </n-list>
        </n-tab-pane>
      </n-tabs>
    </n-space>
  </n-card>
</template>
