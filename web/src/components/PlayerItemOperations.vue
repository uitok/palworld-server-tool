<script setup>
import { computed, ref, watch } from "vue";
import dayjs from "dayjs";
import { useDialog, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import userStore from "@/stores/model/user";
import palItems from "@/assets/items.json";

const props = defineProps({
  playerInfo: {
    type: Object,
    default: () => ({}),
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
  return dayjs().diff(dayjs(lastOnline)) < 80000;
});

const itemAction = ref("grant");
const selectedItemId = ref(null);
const actionAmount = ref(1);
const targetCount = ref(1);
const clearContainers = ref([]);
const submitting = ref(false);

const itemCatalog = computed(() => palItems[locale.value] || palItems.zh || []);
const itemOptions = computed(() =>
  itemCatalog.value.map((item) => ({
    label: `${item.name} (${item.key || item.id})`,
    value: item.key || item.id,
  }))
);

const containerOptions = computed(() => [
  { label: t("item.commonContainer"), value: "CommonContainerId" },
  { label: t("item.essentialContainer"), value: "EssentialContainerId" },
  { label: t("item.weaponContainer"), value: "WeaponLoadOutContainerId" },
  { label: t("item.armorContainer"), value: "PlayerEquipArmorContainerId" },
  { label: t("item.foodContainer"), value: "FoodEquipContainerId" },
  { label: t("item.dropContainer"), value: "DropSlotContainerId" },
]);

const currentTotal = computed(() => {
  if (!selectedItemId.value) {
    return 0;
  }
  const items = props.playerInfo?.items || {};
  return Object.values(items).reduce((total, inventoryItems) => {
    if (!Array.isArray(inventoryItems)) {
      return total;
    }
    return (
      total +
      inventoryItems.reduce((sum, item) => {
        if (item.ItemId !== selectedItemId.value) {
          return sum;
        }
        return sum + Number(item.StackCount || 0);
      }, 0)
    );
  }, 0);
});

const selectedItemDetail = computed(() =>
  itemCatalog.value.find((item) => (item.key || item.id) === selectedItemId.value)
);

const actionButtonLabel = computed(() => {
  if (itemAction.value === "remove") {
    return t("button.remove");
  }
  if (itemAction.value === "set") {
    return t("button.setTarget");
  }
  return t("button.grant");
});

watch(currentTotal, (value) => {
  if (itemAction.value === "set" && value > 0) {
    targetCount.value = value;
  }
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

const submitItemAction = async () => {
  if (!ensureReady()) {
    return;
  }
  if (!selectedItemId.value) {
    message.warning(t("message.selectItemFirst"));
    return;
  }

  submitting.value = true;
  let data;
  let statusCode;
  if (itemAction.value === "grant") {
    ({ data, statusCode } = await new ApiService().grantPlayerItems({
      playerUid: props.playerInfo.player_uid,
      ...buildPlayerPayload(),
      item_id: selectedItemId.value,
      amount: getPositiveNumber(actionAmount.value),
    }));
  } else {
    ({ data, statusCode } = await new ApiService().adjustPlayerItems({
      playerUid: props.playerInfo.player_uid,
      ...buildPlayerPayload(),
      item_id: selectedItemId.value,
      operation: itemAction.value,
      amount: getPositiveNumber(actionAmount.value),
      target_count: getPositiveNumber(targetCount.value),
      current_total: currentTotal.value,
    }));
  }
  submitting.value = false;

  if (statusCode.value === 200) {
    message.success(t("message.itemActionSuccess"));
    return;
  }
  message.error(
    t("message.itemActionFail", {
      err: data.value?.error || data.value?.message || "Unknown error",
    })
  );
};

const confirmClearInventory = () => {
  if (!ensureReady()) {
    return;
  }
  if (!clearContainers.value.length) {
    message.warning(t("message.selectContainerFirst"));
    return;
  }
  dialog.warning({
    title: t("button.clearContainer"),
    content: t("message.clearInventoryWarn"),
    positiveText: t("button.confirm"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      submitting.value = true;
      const { data, statusCode } = await new ApiService().clearPlayerInventory({
        playerUid: props.playerInfo.player_uid,
        ...buildPlayerPayload(),
        containers: clearContainers.value,
      });
      submitting.value = false;
      if (statusCode.value === 200) {
        message.success(t("message.clearInventorySuccess"));
        return;
      }
      message.error(
        t("message.clearInventoryFail", {
          err: data.value?.error || data.value?.message || "Unknown error",
        })
      );
    },
  });
};
</script>

<template>
  <n-card size="small" :bordered="false">
    <template #header>
      {{ $t("button.itemActions") }}
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

      <n-grid :cols="compact ? 1 : 2" :x-gap="12" :y-gap="12">
        <n-gi>
          <n-space vertical>
            <n-select
              v-model:value="selectedItemId"
              filterable
              :placeholder="$t('input.selectItem')"
              :options="itemOptions"
              :disabled="!isPlayerOnline"
            />

            <n-radio-group v-model:value="itemAction" size="small" :disabled="!isPlayerOnline">
              <n-radio-button value="grant">{{ $t("button.grant") }}</n-radio-button>
              <n-radio-button value="remove">{{ $t("button.remove") }}</n-radio-button>
              <n-radio-button value="set">{{ $t("button.setTarget") }}</n-radio-button>
            </n-radio-group>

            <div class="flex items-center gap-2">
              <n-input-number
                v-if="itemAction !== 'set'"
                v-model:value="actionAmount"
                :min="1"
                :precision="0"
                class="flex-1"
                :disabled="!isPlayerOnline"
              />
              <n-input-number
                v-else
                v-model:value="targetCount"
                :min="0"
                :precision="0"
                class="flex-1"
                :disabled="!isPlayerOnline"
              />
              <n-button
                type="primary"
                strong
                secondary
                :loading="submitting"
                :disabled="!isPlayerOnline"
                @click="submitItemAction"
              >
                {{ actionButtonLabel }}
              </n-button>
            </div>

            <n-text depth="3">
              {{ $t("item.currentTotal") }}: {{ currentTotal }}
            </n-text>
            <n-text depth="3" v-if="selectedItemDetail?.description">
              {{ selectedItemDetail.description }}
            </n-text>
          </n-space>
        </n-gi>

        <n-gi>
          <n-space vertical>
            <n-text strong>{{ $t("button.clearContainer") }}</n-text>
            <n-checkbox-group v-model:value="clearContainers" :disabled="!isPlayerOnline">
              <n-space vertical>
                <n-checkbox
                  v-for="container in containerOptions"
                  :key="container.value"
                  :value="container.value"
                  :label="container.label"
                />
              </n-space>
            </n-checkbox-group>
            <n-button
              type="error"
              strong
              secondary
              :loading="submitting"
              :disabled="!isPlayerOnline"
              @click="confirmClearInventory"
            >
              {{ $t("button.clearContainer") }}
            </n-button>
          </n-space>
        </n-gi>
      </n-grid>
    </n-space>
  </n-card>
</template>
