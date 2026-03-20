<script setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import ApiService from "@/service/api";
import userStore from "@/stores/model/user";
import PlayerItemOperations from "@/components/PlayerItemOperations.vue";
import PlayerPalOperations from "@/components/PlayerPalOperations.vue";
import PalDefenderBatchOperations from "@/components/PalDefenderBatchOperations.vue";

const router = useRouter();
const message = useMessage();
const { locale, t } = useI18n();

const players = ref([]);
const guilds = ref([]);
const playerKeyword = ref("");
const playerOnlineOnly = ref(false);
const loadingPlayers = ref(false);
const loadingDetail = ref(false);
const loadingGuilds = ref(false);
const selectedPlayerUid = ref("");
const selectedOverview = ref(null);

const isLogin = computed(() => userStore().getLoginInfo().isLogin);

const texts = computed(() => {
  if (locale.value === "en") {
    return {
      title: "PalDefender Workspace",
      subtitle: "P6 live grant, batch rewards, audit export, and retry console.",
      backHome: "Back Home",
      backOps: "Ops Overview",
      playerWorkspace: "Player Workspace",
      refresh: "Refresh",
      search: "Search",
      authNotice: "Viewing is open, but live PalDefender grant operations still require admin login and online players.",
      playerSelector: "Player Selector",
      playerKeywordPlaceholder: "Search nickname / UID / user id / steam id",
      onlineOnly: "Online only",
      playerCount: "Players",
      selectPlayer: "Select Player",
      noPlayers: "No players found",
      playerDetail: "Live Player Operations",
      selectPlayerHint: "Select a player to open single-player live grant operations.",
      batchTitle: "Batch Rewards & Audit",
      batchSubtitle: "Runs preset-based rewards, audit query/export, and failed batch retry.",
      lastOnline: "Last Online",
      userId: "User ID",
      steamId: "Steam ID",
      guild: "Guild",
      level: "Level",
      whitelist: "Whitelist",
      noData: "No data",
      online: "Online",
      offline: "Offline",
      loadFail: "Failed to load PalDefender workspace data",
    };
  }
  if (locale.value === "ja") {
    return {
      title: "PalDefender ワークスペース",
      subtitle: "P6 のライブ付与、一括報酬、監査エクスポート、失敗再試行の統合画面です。",
      backHome: "ホームへ戻る",
      backOps: "運用ダッシュボード",
      playerWorkspace: "プレイヤーワークスペース",
      refresh: "更新",
      search: "検索",
      authNotice: "閲覧は可能ですが、PalDefender のライブ付与には管理者ログインと対象プレイヤーのオンライン状態が必要です。",
      playerSelector: "プレイヤー選択",
      playerKeywordPlaceholder: "ニックネーム / UID / UserID / SteamID で検索",
      onlineOnly: "オンラインのみ",
      playerCount: "プレイヤー数",
      selectPlayer: "プレイヤーを選択",
      noPlayers: "プレイヤーが見つかりません",
      playerDetail: "単体ライブ操作",
      selectPlayerHint: "プレイヤーを選択すると単体付与操作を開けます。",
      batchTitle: "一括報酬と監査",
      batchSubtitle: "プリセット付与、監査検索/エクスポート、失敗バッチ再試行をここでまとめます。",
      lastOnline: "最終オンライン",
      userId: "User ID",
      steamId: "Steam ID",
      guild: "ギルド",
      level: "レベル",
      whitelist: "ホワイトリスト",
      noData: "データなし",
      online: "オンライン",
      offline: "オフライン",
      loadFail: "PalDefender ワークスペースの読み込みに失敗しました",
    };
  }
  return {
    title: "PalDefender 工作台",
    subtitle: "P6 阶段的单人实时发放、批量礼包、审计导出与失败重试统一入口。",
    backHome: "返回首页",
    backOps: "运维总览",
    playerWorkspace: "玩家工作台",
    refresh: "刷新",
    search: "搜索",
    authNotice: "页面可浏览，但 PalDefender 实时发放仍要求管理员登录且目标玩家在线。",
    playerSelector: "玩家选择",
    playerKeywordPlaceholder: "按昵称 / UID / UserID / SteamID 搜索",
    onlineOnly: "仅在线",
    playerCount: "玩家数",
    selectPlayer: "选择玩家",
    noPlayers: "未找到玩家",
    playerDetail: "单玩家实时操作",
    selectPlayerHint: "请选择玩家后使用单玩家实时发放能力。",
    batchTitle: "批量礼包与审计",
    batchSubtitle: "集中处理预设发放、审计查询/导出，以及失败批次重试。",
    lastOnline: "最近在线",
    userId: "UserID",
    steamId: "SteamID",
    guild: "公会",
    level: "等级",
    whitelist: "白名单",
    noData: "暂无数据",
    online: "在线",
    offline: "离线",
    loadFail: "加载 PalDefender 工作台失败",
  };
});

const selectedSummary = computed(() => selectedOverview.value?.summary || null);
const selectedPlayer = computed(() => selectedOverview.value?.player || null);

const playerOptions = computed(() =>
  players.value.map((player) => ({
    label: `[${player.online ? texts.value.online : texts.value.offline}] ${player.nickname || player.player_uid} (${player.player_uid})`,
    value: player.player_uid,
  }))
);

const formatDateTime = (value) => {
  if (!value) return texts.value.noData;
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format("YYYY-MM-DD HH:mm:ss") : String(value);
};

const fetchGuilds = async () => {
  loadingGuilds.value = true;
  const { data, statusCode } = await new ApiService().getGuildList();
  if (statusCode.value === 200 && Array.isArray(data.value)) {
    guilds.value = data.value;
  } else {
    guilds.value = [];
  }
  loadingGuilds.value = false;
};

const fetchPlayerDetail = async (playerUid, silent = false) => {
  if (!playerUid) {
    selectedPlayerUid.value = "";
    selectedOverview.value = null;
    return;
  }
  selectedPlayerUid.value = playerUid;
  loadingDetail.value = !silent;
  const { data, statusCode } = await new ApiService().getPlayerOverviewDetail({ playerUid });
  if (statusCode.value === 200 && data.value?.overview) {
    selectedOverview.value = data.value.overview;
  } else if (!silent) {
    message.error(texts.value.loadFail);
  }
  loadingDetail.value = false;
};

const fetchPlayers = async (preserveSelection = true) => {
  loadingPlayers.value = true;
  const { data, statusCode } = await new ApiService().getPlayerOverviewList({
    keyword: playerKeyword.value.trim(),
    online_only: playerOnlineOnly.value,
  });
  if (statusCode.value === 200 && data.value?.items) {
    players.value = data.value.items;
    const currentStillExists =
      preserveSelection && players.value.some((player) => player.player_uid === selectedPlayerUid.value);
    if (currentStillExists) {
      await fetchPlayerDetail(selectedPlayerUid.value, true);
    } else if (players.value.length > 0) {
      await fetchPlayerDetail(players.value[0].player_uid, true);
    } else {
      selectedPlayerUid.value = "";
      selectedOverview.value = null;
    }
  } else {
    players.value = [];
    selectedPlayerUid.value = "";
    selectedOverview.value = null;
    message.error(texts.value.loadFail);
  }
  loadingPlayers.value = false;
};

const refreshAll = async (preserveSelection = true) => {
  await Promise.all([fetchPlayers(preserveSelection), fetchGuilds()]);
};

onMounted(async () => {
  await refreshAll(false);
});
</script>

<template>
  <div class="min-h-screen bg-slate-50 dark:bg-slate-950">
    <div class="mx-auto max-w-7xl px-4 py-6">
      <n-space justify="space-between" align="center" class="mb-4" wrap>
        <div>
          <div class="text-2xl font-semibold">{{ texts.title }}</div>
          <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.subtitle }}</div>
        </div>
        <n-space>
          <n-button secondary @click="router.push('/players')">{{ texts.playerWorkspace }}</n-button>
          <n-button secondary @click="router.push('/ops')">{{ texts.backOps }}</n-button>
          <n-button secondary @click="router.push('/')">{{ texts.backHome }}</n-button>
          <n-button
            type="primary"
            secondary
            strong
            :loading="loadingPlayers || loadingGuilds || loadingDetail"
            @click="refreshAll(true)"
          >
            {{ texts.refresh }}
          </n-button>
        </n-space>
      </n-space>

      <n-alert v-if="!isLogin" type="info" class="mb-4">{{ texts.authNotice }}</n-alert>

      <n-grid cols="1 l:3" responsive="screen" x-gap="12" y-gap="12" class="mb-4">
        <n-gi>
          <n-card :title="texts.playerSelector">
            <n-space vertical>
              <n-input
                v-model:value="playerKeyword"
                :placeholder="texts.playerKeywordPlaceholder"
                @keyup.enter="fetchPlayers(false)"
              />
              <n-switch v-model:value="playerOnlineOnly">
                <template #checked>{{ texts.onlineOnly }}</template>
                <template #unchecked>{{ texts.onlineOnly }}</template>
              </n-switch>
              <div class="flex items-center justify-between gap-2">
                <n-text depth="3">{{ texts.playerCount }}: {{ players.length }}</n-text>
                <n-button tertiary size="small" :loading="loadingPlayers" @click="fetchPlayers(false)">
                  {{ texts.search }}
                </n-button>
              </div>
              <n-select
                v-model:value="selectedPlayerUid"
                filterable
                clearable
                :placeholder="texts.selectPlayer"
                :options="playerOptions"
                @update:value="fetchPlayerDetail"
              />
              <n-empty v-if="!loadingPlayers && players.length === 0" :description="texts.noPlayers" />
            </n-space>
          </n-card>
        </n-gi>

        <n-gi span="2">
          <n-card :title="texts.playerDetail">
            <n-spin :show="loadingDetail">
              <n-empty v-if="!selectedSummary || !selectedPlayer" :description="texts.selectPlayerHint" />
              <n-space v-else vertical size="small">
                <div class="flex items-center justify-between gap-3 flex-wrap">
                  <div>
                    <div class="text-lg font-semibold">{{ selectedSummary.nickname || selectedSummary.player_uid }}</div>
                    <div class="text-xs opacity-75">UID: {{ selectedSummary.player_uid }}</div>
                  </div>
                  <n-space size="small">
                    <n-tag :type="selectedSummary.online ? 'success' : 'default'" round>
                      {{ selectedSummary.online ? texts.online : texts.offline }}
                    </n-tag>
                    <n-tag v-if="selectedSummary.whitelisted" type="warning" round>
                      {{ texts.whitelist }}
                    </n-tag>
                    <n-tag v-if="selectedSummary.guild" type="info" round>
                      {{ selectedSummary.guild.name }}
                    </n-tag>
                  </n-space>
                </div>

                <n-grid cols="1 s:2 l:4" responsive="screen" x-gap="12" y-gap="12">
                  <n-gi>
                    <n-card size="small">
                      <n-statistic :label="texts.level" :value="selectedSummary.level || 0" />
                    </n-card>
                  </n-gi>
                  <n-gi>
                    <n-card size="small">
                      <n-statistic :label="texts.lastOnline" :value="formatDateTime(selectedSummary.last_online)" />
                    </n-card>
                  </n-gi>
                  <n-gi>
                    <n-card size="small">
                      <n-statistic :label="texts.userId" :value="selectedSummary.user_id || texts.noData" />
                    </n-card>
                  </n-gi>
                  <n-gi>
                    <n-card size="small">
                      <n-statistic :label="texts.steamId" :value="selectedSummary.steam_id || texts.noData" />
                      <template #footer>
                        <div class="text-xs opacity-75">{{ texts.guild }}: {{ selectedSummary.guild?.name || texts.noData }}</div>
                      </template>
                    </n-card>
                  </n-gi>
                </n-grid>

                <n-grid cols="1 l:2" responsive="screen" x-gap="12" y-gap="12">
                  <n-gi>
                    <player-item-operations :player-info="selectedPlayer" compact />
                  </n-gi>
                  <n-gi>
                    <player-pal-operations
                      :player-info="selectedPlayer"
                      :player-pals-list="selectedPlayer.pals || []"
                      compact
                    />
                  </n-gi>
                </n-grid>
              </n-space>
            </n-spin>
          </n-card>
        </n-gi>
      </n-grid>

      <div class="text-lg font-semibold mb-2">{{ texts.batchTitle }}</div>
      <div class="text-sm text-slate-500 dark:text-slate-300 mb-3">{{ texts.batchSubtitle }}</div>
      <pal-defender-batch-operations :player-list="players" :guild-list="guilds" />
    </div>
  </div>
</template>
