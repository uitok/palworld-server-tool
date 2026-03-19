<script setup>
import { computed, ref } from "vue";
import dayjs from "dayjs";
import { useDialog, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import userStore from "@/stores/model/user";
import palMap from "@/assets/pal.json";
import palItems from "@/assets/items.json";
import { explainPalDefenderError, isPlayerOnlineForLiveGrant } from "@/utils/paldefender";

const props = defineProps({
  playerInfo: {
    type: Object,
    default: () => ({}),
  },
  playerPalsList: {
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
const dialog = useDialog();

const isLogin = computed(() => userStore().getLoginInfo().isLogin);
const isPlayerOnline = computed(() => {
  const lastOnline = props.playerInfo?.last_online;
  if (!lastOnline) {
    return false;
  }
  return isPlayerOnlineForLiveGrant(lastOnline);
});

const grantPalId = ref(null);
const grantPalLevel = ref(1);
const grantPalAmount = ref(1);
const grantEggId = ref(null);
const grantEggPalId = ref(null);
const grantEggLevel = ref(1);
const grantEggAmount = ref(1);
const templateName = ref("");
const templateAmount = ref(1);
const deletePalId = ref(null);
const deleteNickname = ref("");
const deleteGender = ref("");
const deleteLucky = ref("");
const deleteLevelCompare = ref("gte");
const deleteLevelValue = ref(null);
const deleteRankCompare = ref("eq");
const deleteRankValue = ref(null);
const deletePassives = ref("");
const deleteLimit = ref(0);
const submitting = ref(false);

const palOptions = computed(() =>
  Object.entries(palMap[locale.value] || palMap.zh || {}).map(([palId, palName]) => ({
    label: `${palName} (${palId})`,
    value: palId,
  }))
);

const eggOptions = computed(() =>
  (palItems[locale.value] || palItems.zh || [])
    .filter((item) => /^PalEgg_/i.test(item.key || ""))
    .map((item) => ({
      label: `${item.name} (${item.key || item.id})`,
      value: item.key || item.id,
    }))
);

const genderOptions = computed(() => [
  { label: t("pal.anyGender"), value: "" },
  { label: t("pal.male"), value: "Male" },
  { label: t("pal.female"), value: "Female" },
]);

const luckyOptions = computed(() => [
  { label: t("pal.anyLucky"), value: "" },
  { label: t("pal.luckyOnly"), value: "true" },
  { label: t("pal.nonLuckyOnly"), value: "false" },
]);

const compareOptions = [
  { label: ">=", value: "gte" },
  { label: "=", value: "eq" },
  { label: "<=", value: "lte" },
];

const currentPalCount = computed(() => props.playerPalsList?.length || 0);

const ensureReady = () => {
  if (!isLogin.value) {
    message.error(t("message.requireauth"));
    return false;
  }
  if (!props.playerInfo?.player_uid) {
    message.error(t("message.playerRequired"));
    return false;
  }
  if (!isPlayerOnline.value) {
    message.warning(t("message.playerMustBeOnline"));
    return false;
  }
  return true;
};

const buildPlayerPayload = () => ({
  player_uid: props.playerInfo?.player_uid,
  user_id: props.playerInfo?.user_id || "",
  steam_id: props.playerInfo?.steam_id || "",
});

const getPositiveNumber = (value, fallback = 1, allowZero = false) => {
  const parsed = Number.parseInt(String(value ?? fallback), 10);
  if (!Number.isFinite(parsed)) {
    return fallback;
  }
  if (allowZero && parsed === 0) {
    return 0;
  }
  return parsed > 0 ? parsed : fallback;
};

const buildDeleteFilters = () => {
  const filters = {};
  if (deletePalId.value) {
    filters.pal_id = deletePalId.value;
  }
  if (deleteNickname.value.trim()) {
    filters.nickname = deleteNickname.value.trim();
  }
  if (deleteGender.value) {
    filters.gender = deleteGender.value;
  }
  if (deleteLucky.value !== "") {
    filters.is_lucky = deleteLucky.value === "true";
  }
  if (deleteLevelValue.value) {
    filters.level_compare = deleteLevelCompare.value;
    filters.level = getPositiveNumber(deleteLevelValue.value);
  }
  if (deleteRankValue.value !== null && deleteRankValue.value !== undefined) {
    filters.rank_compare = deleteRankCompare.value;
    filters.rank = getPositiveNumber(deleteRankValue.value, 1, true);
  }
  const passiveKeywords = deletePassives.value
    .split(",")
    .map((keyword) => keyword.trim())
    .filter(Boolean);
  if (passiveKeywords.length) {
    filters.passive_keywords = passiveKeywords;
  }
  return filters;
};

const deletePreview = computed(() => {
  const filters = buildDeleteFilters();
  const preview = [];
  if (filters.pal_id) {
    preview.push(`pal=${filters.pal_id}`);
  }
  if (filters.nickname) {
    preview.push(`nickname~${filters.nickname}`);
  }
  if (filters.gender) {
    preview.push(`gender=${filters.gender}`);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "is_lucky")) {
    preview.push(`lucky=${filters.is_lucky}`);
  }
  if (filters.level) {
    preview.push(`level ${filters.level_compare} ${filters.level}`);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "rank")) {
    preview.push(`rank ${filters.rank_compare} ${filters.rank}`);
  }
  if (filters.passive_keywords?.length) {
    preview.push(`passives=${filters.passive_keywords.join("|")}`);
  }
  if (deleteLimit.value) {
    preview.push(`limit=${deleteLimit.value}`);
  }
  return preview.length ? preview.join(" · ") : "-";
});

const submitGrantPal = async () => {
  if (!ensureReady()) {
    return;
  }
  if (!grantPalId.value) {
    message.warning(t("message.selectPalFirst"));
    return;
  }

  submitting.value = true;
  const { data, statusCode } = await new ApiService().grantPlayerPal({
    playerUid: props.playerInfo.player_uid,
    ...buildPlayerPayload(),
    pal_id: grantPalId.value,
    level: getPositiveNumber(grantPalLevel.value),
    amount: getPositiveNumber(grantPalAmount.value),
  });
  submitting.value = false;

  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(
    t("message.palActionFail", {
      err: explainPalDefenderError(t, data.value),
    })
  );
};

const submitGrantTemplate = async () => {
  if (!ensureReady()) {
    return;
  }
  if (!templateName.value.trim()) {
    message.warning(t("message.templateNameRequired"));
    return;
  }

  submitting.value = true;
  const { data, statusCode } = await new ApiService().grantPlayerPalTemplate({
    playerUid: props.playerInfo.player_uid,
    ...buildPlayerPayload(),
    template_name: templateName.value.trim(),
    amount: getPositiveNumber(templateAmount.value),
  });
  submitting.value = false;

  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(
    t("message.palActionFail", {
      err: explainPalDefenderError(t, data.value),
    })
  );
};

const submitGrantEgg = async () => {
  if (!ensureReady()) {
    return;
  }
  if (!grantEggId.value) {
    message.warning(t("message.selectEggFirst"));
    return;
  }
  if (!grantEggPalId.value) {
    message.warning(t("message.selectPalFirst"));
    return;
  }

  submitting.value = true;
  const { data, statusCode } = await new ApiService().grantPlayerPalEgg({
    playerUid: props.playerInfo.player_uid,
    ...buildPlayerPayload(),
    egg_id: grantEggId.value,
    pal_id: grantEggPalId.value,
    level: getPositiveNumber(grantEggLevel.value),
    amount: getPositiveNumber(grantEggAmount.value),
  });
  submitting.value = false;

  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(
    t("message.palActionFail", {
      err: explainPalDefenderError(t, data.value),
    })
  );
};

const submitExportPals = async () => {
  if (!ensureReady()) {
    return;
  }

  submitting.value = true;
  const { data, statusCode } = await new ApiService().exportPlayerPals({
    playerUid: props.playerInfo.player_uid,
    ...buildPlayerPayload(),
  });
  submitting.value = false;

  if (statusCode.value === 200) {
    message.success(t("message.exportPalsSuccess"));
    return;
  }
  message.error(
    t("message.exportPalsFail", {
      err: explainPalDefenderError(t, data.value),
    })
  );
};

const submitDeletePals = () => {
  if (!ensureReady()) {
    return;
  }

  const filters = buildDeleteFilters();
  if (!Object.keys(filters).length) {
    message.warning(t("message.deleteFilterRequired"));
    return;
  }

  dialog.warning({
    title: t("button.deletePals"),
    content: deletePreview.value,
    positiveText: t("button.confirm"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      submitting.value = true;
      const { data, statusCode } = await new ApiService().deletePlayerPals({
        playerUid: props.playerInfo.player_uid,
        ...buildPlayerPayload(),
        filters,
        limit: getPositiveNumber(deleteLimit.value, 0, true),
      });
      submitting.value = false;
      if (statusCode.value === 200) {
        message.success(t("message.deletePalsSuccess"));
        return;
      }
      message.error(
        t("message.deletePalsFail", {
          err: explainPalDefenderError(t, data.value),
        })
      );
    },
  });
};
</script>

<template>
  <n-card size="small" :bordered="false">
    <template #header>
      {{ $t("button.palActions") }}
    </template>
    <template #header-extra>
      <n-tag type="warning" round>PalDefender</n-tag>
    </template>

    <n-space vertical size="small">
      <n-alert type="info" :show-icon="false">
        {{ $t("message.palDefenderUiHint") }}
      </n-alert>
      <n-alert v-if="!isPlayerOnline" type="warning" :show-icon="false">
        {{ $t("message.playerOfflineActionHint") }}
      </n-alert>

      <div class="flex items-center justify-between gap-2 flex-wrap">
        <n-text depth="3">{{ $t("pal.currentCount") }}: {{ currentPalCount }}</n-text>
        <n-text depth="3">UID: {{ playerInfo?.player_uid || "-" }}</n-text>
      </div>

      <n-tabs type="line" animated>
        <n-tab-pane name="grant" :tab="$t('button.grant')">
          <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
            <n-gi>
              <n-space vertical>
                <n-select
                  v-model:value="grantPalId"
                  filterable
                  :placeholder="$t('input.selectPal')"
                  :options="palOptions"
                  :disabled="!isPlayerOnline"
                />
                <div class="flex gap-2">
                  <n-input-number
                    v-model:value="grantPalLevel"
                    :min="1"
                    :precision="0"
                    class="flex-1"
                    :placeholder="$t('pal.level')"
                    :disabled="!isPlayerOnline"
                  />
                  <n-input-number
                    v-model:value="grantPalAmount"
                    :min="1"
                    :precision="0"
                    class="flex-1"
                    :placeholder="$t('input.amount')"
                    :disabled="!isPlayerOnline"
                  />
                </div>
                <n-button
                  type="primary"
                  strong
                  secondary
                  :loading="submitting"
                  :disabled="!isPlayerOnline"
                  @click="submitGrantPal"
                >
                  {{ $t("button.quickGivePal") }}
                </n-button>
              </n-space>
            </n-gi>

            <n-gi>
              <n-space vertical>
                <n-select
                  v-model:value="grantEggId"
                  filterable
                  :placeholder="$t('input.selectEgg')"
                  :options="eggOptions"
                  :disabled="!isPlayerOnline"
                />
                <n-select
                  v-model:value="grantEggPalId"
                  filterable
                  :placeholder="$t('input.selectPal')"
                  :options="palOptions"
                  :disabled="!isPlayerOnline"
                />
                <div class="flex gap-2">
                  <n-input-number
                    v-model:value="grantEggLevel"
                    :min="1"
                    :precision="0"
                    class="flex-1"
                    :placeholder="$t('pal.level')"
                    :disabled="!isPlayerOnline"
                  />
                  <n-input-number
                    v-model:value="grantEggAmount"
                    :min="1"
                    :precision="0"
                    class="flex-1"
                    :placeholder="$t('input.amount')"
                    :disabled="!isPlayerOnline"
                  />
                </div>
                <n-button
                  type="warning"
                  strong
                  secondary
                  :loading="submitting"
                  :disabled="!isPlayerOnline"
                  @click="submitGrantEgg"
                >
                  {{ $t("button.quickGiveEgg") }}
                </n-button>

                <n-input
                  v-model:value="templateName"
                  :placeholder="$t('input.templateName')"
                  :disabled="!isPlayerOnline"
                />
                <div class="flex gap-2">
                  <n-input-number
                    v-model:value="templateAmount"
                    :min="1"
                    :precision="0"
                    class="flex-1"
                    :placeholder="$t('input.amount')"
                    :disabled="!isPlayerOnline"
                  />
                  <n-button
                    class="flex-1"
                    type="success"
                    strong
                    secondary
                    :loading="submitting"
                    :disabled="!isPlayerOnline"
                    @click="submitGrantTemplate"
                  >
                    {{ $t("button.grantTemplate") }}
                  </n-button>
                </div>
                <n-button
                  type="info"
                  strong
                  secondary
                  :loading="submitting"
                  :disabled="!isPlayerOnline"
                  @click="submitExportPals"
                >
                  {{ $t("button.exportPals") }}
                </n-button>
              </n-space>
            </n-gi>
          </n-grid>
        </n-tab-pane>

        <n-tab-pane name="delete" :tab="$t('button.deletePals')">
          <n-space vertical>
            <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
              <n-gi>
                <n-space vertical>
                  <n-select
                    v-model:value="deletePalId"
                    filterable
                    clearable
                    :placeholder="$t('input.selectPal')"
                    :options="palOptions"
                    :disabled="!isPlayerOnline"
                  />
                  <n-input
                    v-model:value="deleteNickname"
                    clearable
                    :placeholder="$t('input.nicknameFilter')"
                    :disabled="!isPlayerOnline"
                  />
                  <div class="flex gap-2">
                    <n-select
                      v-model:value="deleteGender"
                      class="flex-1"
                      :placeholder="$t('input.gender')"
                      :options="genderOptions"
                      :disabled="!isPlayerOnline"
                    />
                    <n-select
                      v-model:value="deleteLucky"
                      class="flex-1"
                      :options="luckyOptions"
                      :disabled="!isPlayerOnline"
                    />
                  </div>
                </n-space>
              </n-gi>

              <n-gi>
                <n-space vertical>
                  <div class="flex gap-2">
                    <n-select
                      v-model:value="deleteLevelCompare"
                      class="w-32"
                      :placeholder="$t('input.levelCompare')"
                      :options="compareOptions"
                      :disabled="!isPlayerOnline"
                    />
                    <n-input-number
                      v-model:value="deleteLevelValue"
                      :min="1"
                      :precision="0"
                      class="flex-1"
                      :disabled="!isPlayerOnline"
                    />
                  </div>
                  <div class="flex gap-2">
                    <n-select
                      v-model:value="deleteRankCompare"
                      class="w-32"
                      :placeholder="$t('input.rankCompare')"
                      :options="compareOptions"
                      :disabled="!isPlayerOnline"
                    />
                    <n-input-number
                      v-model:value="deleteRankValue"
                      :min="0"
                      :precision="0"
                      class="flex-1"
                      :disabled="!isPlayerOnline"
                    />
                  </div>
                  <n-input
                    v-model:value="deletePassives"
                    clearable
                    :placeholder="$t('input.passiveKeywords')"
                    :disabled="!isPlayerOnline"
                  />
                  <n-input-number
                    v-model:value="deleteLimit"
                    :min="0"
                    :precision="0"
                    :placeholder="$t('input.limit')"
                    :disabled="!isPlayerOnline"
                  />
                </n-space>
              </n-gi>
            </n-grid>

            <n-alert type="warning" :show-icon="false">
              {{ $t("pal.deletePreview") }}: {{ deletePreview }}
            </n-alert>

            <n-button
              type="error"
              strong
              secondary
              :loading="submitting"
              :disabled="!isPlayerOnline"
              @click="submitDeletePals"
            >
              {{ $t("button.deletePals") }}
            </n-button>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-space>
  </n-card>
</template>
