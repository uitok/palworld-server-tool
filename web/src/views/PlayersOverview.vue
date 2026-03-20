<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useDialog, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import ApiService from "@/service/api";
import userStore from "@/stores/model/user";
import playerToGuildStore from "@/stores/model/playerToGuild";
import MapView from "@/views/PcHome/component/MapView.vue";

const router = useRouter();
const message = useMessage();
const dialog = useDialog();
const { locale } = useI18n();
const store = userStore();
const linkStore = playerToGuildStore();

const loading = ref(false);
const detailLoading = ref(false);
const guildLoading = ref(false);
const itemSearching = ref(false);
const palSearching = ref(false);
const batchLoading = ref(false);
const activeTab = ref("players");
const filters = ref({
  keyword: "",
  online_only: false,
  whitelist_only: false,
  guild_only: false,
});
const players = ref([]);
const guilds = ref([]);
const guildKeyword = ref("");
const selectedPlayerUid = ref("");
const selectedOverview = ref(null);
const selectedPlayerUids = ref([]);
const itemKeyword = ref("");
const palKeyword = ref("");
const searchSelectedPlayerOnly = ref(false);
const itemResults = ref([]);
const palResults = ref([]);

const isLogin = computed(() => store.getLoginInfo().isLogin);

const texts = computed(() => {
  if (locale.value === "en") {
    return {
      title: "Player Workspace",
      subtitle: "P5 player management, guild/map search, and offline data workspace.",
      backHome: "Back Home",
      backOps: "Ops Overview",
      palDefenderWorkspace: "PalDefender Workspace",
      playersTab: "Players",
      itemsTab: "Items Search",
      palsTab: "Pals Search",
      guildsTab: "Guild Search",
      mapTab: "Map",
      authNotice: "Management actions and global item/pal search require admin login.",
      playerFilters: "Player Filters",
      keywordPlaceholder: "Search by nickname / UID / user id / steam id / guild",
      onlineOnly: "Online only",
      whitelistOnly: "Whitelist only",
      guildOnly: "Guild only",
      refresh: "Refresh",
      totalPlayers: "Total Players",
      onlinePlayers: "Online",
      whitelistPlayers: "Whitelist",
      guildPlayers: "With Guild",
      playerList: "Player List",
      playerDetail: "Player Detail",
      selectPlayer: "Select a player to view aggregated detail.",
      identifiers: "Identifiers",
      accountName: "Account Name",
      level: "Level",
      lastOnline: "Last Online",
      buildingCount: "Buildings",
      coordinates: "Coordinates",
      guild: "Guild",
      whitelist: "Whitelist",
      online: "Online",
      offline: "Offline",
      topItems: "Top Items",
      palPreview: "Pal Preview",
      noData: "No data",
      searchKeyword: "Keyword",
      searchCurrentOnly: "Selected player only",
      searchItems: "Search Items",
      searchPals: "Search Pals",
      itemCount: "Count",
      container: "Container",
      player: "Player",
      palNickname: "Nickname",
      palId: "Pal ID",
      palSkills: "Skills",
      notAvailable: "Unavailable",
      loadFail: "Failed to load data",
      searchFail: "Search failed",
      keywordRequired: "Keyword is required",
      copied: "Copied",
      copyFail: "Copy failed",
      playerActions: "Player Actions",
      addWhitelist: "Add Whitelist",
      removeWhitelist: "Remove Whitelist",
      kick: "Kick",
      ban: "Ban",
      unban: "Unban",
      selectedCount: "Selected",
      selectVisible: "Select Visible",
      clearSelection: "Clear Selection",
      batchActions: "Batch Actions",
      batchCompleted: "Batch action completed",
      guildKeywordPlaceholder: "Search guild / admin UID / member / coordinates",
      guildName: "Guild Name",
      memberCount: "Members",
      baseCamp: "Base Camps",
      viewPlayer: "View Player",
      confirmAction: "Confirm Action",
      confirmBatchAction: "Confirm Batch Action",
      batchTargetsEmpty: "Select at least one player",
      actionSuccess: "Action completed",
      actionFail: "Action failed",
      resultSummary: "Result Summary",
      mapHint: "Use the map to inspect current players and guild base camps.",
    };
  }
  if (locale.value === "ja") {
    return {
      title: "プレイヤーワークスペース",
      subtitle: "P5 のプレイヤー管理・ギルド/マップ検索・オフラインデータ作業台です。",
      backHome: "ホームへ戻る",
      backOps: "運用ダッシュボード",
      palDefenderWorkspace: "PalDefender ワークスペース",
      playersTab: "プレイヤー",
      itemsTab: "アイテム検索",
      palsTab: "パル検索",
      guildsTab: "ギルド検索",
      mapTab: "マップ",
      authNotice: "管理操作とグローバルなアイテム/パル検索には管理者ログインが必要です。",
      playerFilters: "プレイヤーフィルター",
      keywordPlaceholder: "ニックネーム / UID / UserID / SteamID / ギルドで検索",
      onlineOnly: "オンラインのみ",
      whitelistOnly: "ホワイトリストのみ",
      guildOnly: "ギルドありのみ",
      refresh: "更新",
      totalPlayers: "総プレイヤー数",
      onlinePlayers: "オンライン",
      whitelistPlayers: "ホワイトリスト",
      guildPlayers: "ギルドあり",
      playerList: "プレイヤー一覧",
      playerDetail: "プレイヤー詳細",
      selectPlayer: "左のプレイヤーを選択して集約詳細を表示します。",
      identifiers: "識別情報",
      accountName: "アカウント名",
      level: "レベル",
      lastOnline: "最終オンライン",
      buildingCount: "建築数",
      coordinates: "座標",
      guild: "ギルド",
      whitelist: "ホワイトリスト",
      online: "オンライン",
      offline: "オフライン",
      topItems: "主要アイテム",
      palPreview: "パル概要",
      noData: "データなし",
      searchKeyword: "キーワード",
      searchCurrentOnly: "選択中プレイヤーのみ",
      searchItems: "アイテム検索",
      searchPals: "パル検索",
      itemCount: "数",
      container: "コンテナ",
      player: "プレイヤー",
      palNickname: "ニックネーム",
      palId: "パル ID",
      palSkills: "スキル",
      notAvailable: "未取得",
      loadFail: "読み込みに失敗しました",
      searchFail: "検索に失敗しました",
      keywordRequired: "キーワードを入力してください",
      copied: "コピーしました",
      copyFail: "コピーに失敗しました",
      playerActions: "プレイヤー操作",
      addWhitelist: "ホワイトリスト追加",
      removeWhitelist: "ホワイトリスト削除",
      kick: "キック",
      ban: "BAN",
      unban: "BAN解除",
      selectedCount: "選択数",
      selectVisible: "表示中を選択",
      clearSelection: "選択解除",
      batchActions: "一括操作",
      batchCompleted: "一括操作が完了しました",
      guildKeywordPlaceholder: "ギルド / 管理 UID / メンバー / 座標で検索",
      guildName: "ギルド名",
      memberCount: "人数",
      baseCamp: "拠点",
      viewPlayer: "プレイヤーを見る",
      confirmAction: "操作確認",
      confirmBatchAction: "一括操作確認",
      batchTargetsEmpty: "少なくとも 1 人選択してください",
      actionSuccess: "操作が完了しました",
      actionFail: "操作に失敗しました",
      resultSummary: "結果概要",
      mapHint: "マップで現在のプレイヤー位置とギルド拠点を確認できます。",
    };
  }
  return {
    title: "玩家工作台",
    subtitle: "P5 阶段的玩家管理、工会/地图检索与离线数据工作台。",
    backHome: "返回首页",
    backOps: "运维总览",
    palDefenderWorkspace: "PalDefender 工作台",
    playersTab: "玩家总览",
    itemsTab: "物品检索",
    palsTab: "帕鲁检索",
    guildsTab: "公会检索",
    mapTab: "地图",
    authNotice: "管理动作和全局物品/帕鲁搜索需要管理员登录。",
    playerFilters: "玩家筛选",
    keywordPlaceholder: "按昵称 / UID / UserID / SteamID / 公会搜索",
    onlineOnly: "仅在线",
    whitelistOnly: "仅白名单",
    guildOnly: "仅有公会",
    refresh: "刷新",
    totalPlayers: "玩家总数",
    onlinePlayers: "在线人数",
    whitelistPlayers: "白名单",
    guildPlayers: "有公会",
    playerList: "玩家列表",
    playerDetail: "玩家详情",
    selectPlayer: "请选择左侧玩家查看聚合详情。",
    identifiers: "标识信息",
    accountName: "账号名",
    level: "等级",
    lastOnline: "最近在线",
    buildingCount: "建筑数",
    coordinates: "坐标",
    guild: "公会",
    whitelist: "白名单",
    online: "在线",
    offline: "离线",
    topItems: "物品预览",
    palPreview: "帕鲁预览",
    noData: "暂无数据",
    searchKeyword: "关键词",
    searchCurrentOnly: "仅当前选中玩家",
    searchItems: "搜索物品",
    searchPals: "搜索帕鲁",
    itemCount: "数量",
    container: "容器",
    player: "玩家",
    palNickname: "昵称",
    palId: "帕鲁 ID",
    palSkills: "技能",
    notAvailable: "未获取",
    loadFail: "加载失败",
    searchFail: "搜索失败",
    keywordRequired: "请输入关键词",
    copied: "已复制",
    copyFail: "复制失败",
    playerActions: "玩家操作",
    addWhitelist: "加入白名单",
    removeWhitelist: "移出白名单",
    kick: "踢出",
    ban: "封禁",
    unban: "解封",
    selectedCount: "已选人数",
    selectVisible: "选择当前列表",
    clearSelection: "清空选择",
    batchActions: "批量动作",
    batchCompleted: "批量动作已完成",
    guildKeywordPlaceholder: "按公会 / 管理 UID / 成员 / 坐标搜索",
    guildName: "公会名称",
    memberCount: "成员数",
    baseCamp: "据点",
    viewPlayer: "查看玩家",
    confirmAction: "确认操作",
    confirmBatchAction: "确认批量操作",
    batchTargetsEmpty: "请至少选择一名玩家",
    actionSuccess: "操作完成",
    actionFail: "操作失败",
    resultSummary: "结果摘要",
    mapHint: "可以通过地图查看当前玩家位置和工会据点。",
  };
});

const stats = computed(() => ({
  total: players.value.length,
  online: players.value.filter((item) => item.online).length,
  whitelist: players.value.filter((item) => item.whitelisted).length,
  guild: players.value.filter((item) => item.guild).length,
}));

const selectedSummary = computed(() => selectedOverview.value?.summary || null);
const selectedPlayer = computed(() => selectedOverview.value?.player || null);
const selectedUIDSet = computed(() => new Set(selectedPlayerUids.value));
const filteredGuilds = computed(() => {
  const keyword = guildKeyword.value.trim().toLowerCase();
  if (!keyword) return guilds.value;
  return guilds.value.filter((guild) => {
    const fields = [guild.name, guild.admin_player_uid, String(guild.base_camp_level)];
    (guild.players || []).forEach((player) => {
      fields.push(player.nickname, player.player_uid);
    });
    (guild.base_camp || []).forEach((camp) => {
      fields.push(String(camp.location_x), String(camp.location_y), camp.id);
    });
    return fields.some((field) => String(field || "").toLowerCase().includes(keyword));
  });
});
const allVisibleSelected = computed(() => {
  if (players.value.length === 0) return false;
  return players.value.every((item) => selectedUIDSet.value.has(item.player_uid));
});

const topItems = computed(() => {
  const containers = selectedPlayer.value?.items || {};
  const merged = [];
  Object.entries(containers).forEach(([container, items]) => {
    (items || []).forEach((item) => {
      if (!item) return;
      merged.push({
        key: `${container}-${item.ItemId}-${item.SlotIndex}`,
        container,
        itemId: item.ItemId,
        stackCount: item.StackCount,
      });
    });
  });
  return merged.sort((a, b) => b.stackCount - a.stackCount).slice(0, 8);
});

const topPals = computed(() => {
  return (selectedPlayer.value?.pals || [])
    .filter(Boolean)
    .slice()
    .sort((a, b) => b.level - a.level)
    .slice(0, 8);
});

const scopePlayerUid = computed(() =>
  searchSelectedPlayerOnly.value ? selectedPlayerUid.value : ""
);

const fetchPlayers = async (preserveSelection = true) => {
  loading.value = true;
  const { data, statusCode } = await new ApiService().getPlayerOverviewList(filters.value);
  if (statusCode.value === 200 && data.value?.items) {
    players.value = data.value.items;
    selectedPlayerUids.value = selectedPlayerUids.value.filter((uid) =>
      players.value.some((item) => item.player_uid === uid)
    );
    const stillExists =
      preserveSelection && players.value.some((item) => item.player_uid === selectedPlayerUid.value);
    if (stillExists) {
      await fetchPlayerDetail(selectedPlayerUid.value, true);
    } else if (players.value.length > 0) {
      await fetchPlayerDetail(players.value[0].player_uid, true);
    } else {
      selectedPlayerUid.value = "";
      selectedOverview.value = null;
    }
  } else {
    message.error(texts.value.loadFail);
  }
  loading.value = false;
};

const fetchGuilds = async () => {
  guildLoading.value = true;
  const { data, statusCode } = await new ApiService().getGuildList();
  if (statusCode.value === 200 && Array.isArray(data.value)) {
    guilds.value = data.value;
  } else {
    guilds.value = [];
  }
  guildLoading.value = false;
};

const fetchPlayerDetail = async (playerUid, silent = false) => {
  if (!playerUid) return;
  selectedPlayerUid.value = playerUid;
  detailLoading.value = !silent;
  const { data, statusCode } = await new ApiService().getPlayerOverviewDetail({ playerUid });
  if (statusCode.value === 200 && data.value?.overview) {
    selectedOverview.value = data.value.overview;
  } else if (!silent) {
    message.error(texts.value.loadFail);
  }
  detailLoading.value = false;
};

const refreshAll = async (preserveSelection = true) => {
  await Promise.all([fetchPlayers(preserveSelection), fetchGuilds()]);
};

const ensureKeyword = (value) => {
  if (String(value || "").trim()) return true;
  message.warning(texts.value.keywordRequired);
  return false;
};

const ensureAuth = () => {
  if (isLogin.value) return true;
  message.warning(texts.value.authNotice);
  return false;
};

const togglePlayerSelection = (playerUid, checked) => {
  if (checked) {
    if (!selectedUIDSet.value.has(playerUid)) {
      selectedPlayerUids.value = [...selectedPlayerUids.value, playerUid];
    }
    return;
  }
  selectedPlayerUids.value = selectedPlayerUids.value.filter((uid) => uid !== playerUid);
};

const toggleSelectVisible = () => {
  if (allVisibleSelected.value) {
    selectedPlayerUids.value = selectedPlayerUids.value.filter(
      (uid) => !players.value.some((item) => item.player_uid === uid)
    );
    return;
  }
  const merged = new Set(selectedPlayerUids.value);
  players.value.forEach((item) => merged.add(item.player_uid));
  selectedPlayerUids.value = Array.from(merged);
};

const performSingleAction = (action) => {
  if (!ensureAuth() || !selectedSummary.value) return;
  const actionLabel = {
    whitelist_add: texts.value.addWhitelist,
    whitelist_remove: texts.value.removeWhitelist,
    kick: texts.value.kick,
    ban: texts.value.ban,
    unban: texts.value.unban,
  }[action] || action;
  dialog.warning({
    title: texts.value.confirmAction,
    content: `${actionLabel} · ${selectedSummary.value.nickname || selectedSummary.value.player_uid}`,
    positiveText: actionLabel,
    negativeText: "Cancel",
    onPositiveClick: async () => {
      let result;
      const player = {
        name: selectedSummary.value.nickname,
        player_uid: selectedSummary.value.player_uid,
        steam_id: selectedSummary.value.steam_id,
      };
      if (action === "whitelist_add") {
        result = await new ApiService().addWhitelist(player);
      } else if (action === "whitelist_remove") {
        result = await new ApiService().removeWhitelist(player);
      } else if (action === "kick") {
        result = await new ApiService().kickPlayer({ playerUid: selectedSummary.value.player_uid });
      } else if (action === "ban") {
        result = await new ApiService().banPlayer({ playerUid: selectedSummary.value.player_uid });
      } else if (action === "unban") {
        result = await new ApiService().unbanPlayer({ playerUid: selectedSummary.value.player_uid });
      }
      if (result?.statusCode?.value === 200) {
        message.success(texts.value.actionSuccess);
        await fetchPlayers(true);
      } else {
        message.error(result?.data?.value?.error || texts.value.actionFail);
      }
    },
  });
};

const performBatchAction = (action) => {
  if (!ensureAuth()) return;
  if (selectedPlayerUids.value.length === 0) {
    message.warning(texts.value.batchTargetsEmpty);
    return;
  }
  const actionLabel = {
    whitelist_add: texts.value.addWhitelist,
    whitelist_remove: texts.value.removeWhitelist,
    kick: texts.value.kick,
    ban: texts.value.ban,
    unban: texts.value.unban,
  }[action] || action;
  dialog.warning({
    title: texts.value.confirmBatchAction,
    content: `${actionLabel} · ${texts.value.selectedCount}: ${selectedPlayerUids.value.length}`,
    positiveText: actionLabel,
    negativeText: "Cancel",
    onPositiveClick: async () => {
      batchLoading.value = true;
      const { data, statusCode } = await new ApiService().batchPlayerAction({
        action,
        player_uids: selectedPlayerUids.value,
      });
      if (statusCode.value === 200 && data.value) {
        const result = data.value;
        if (result.failed > 0) {
          message.warning(`${texts.value.batchCompleted}: ${result.succeeded}/${result.requested}`);
        } else {
          message.success(`${texts.value.batchCompleted}: ${result.succeeded}/${result.requested}`);
        }
        await fetchPlayers(true);
      } else {
        message.error(texts.value.actionFail);
      }
      batchLoading.value = false;
    },
  });
};

const runItemSearch = async () => {
  if (!ensureAuth() || !ensureKeyword(itemKeyword.value)) return;
  itemSearching.value = true;
  const { data, statusCode } = await new ApiService().searchPlayerItems({
    keyword: itemKeyword.value.trim(),
    player_uid: scopePlayerUid.value,
  });
  if (statusCode.value === 200 && data.value?.items) {
    itemResults.value = data.value.items;
  } else {
    message.error(texts.value.searchFail);
  }
  itemSearching.value = false;
};

const runPalSearch = async () => {
  if (!ensureAuth() || !ensureKeyword(palKeyword.value)) return;
  palSearching.value = true;
  const { data, statusCode } = await new ApiService().searchPlayerPals({
    keyword: palKeyword.value.trim(),
    player_uid: scopePlayerUid.value,
  });
  if (statusCode.value === 200 && data.value?.items) {
    palResults.value = data.value.items;
  } else {
    message.error(texts.value.searchFail);
  }
  palSearching.value = false;
};

const openGuildPlayer = async (playerUid) => {
  activeTab.value = "players";
  await fetchPlayerDetail(playerUid);
};

const formatDateTime = (value) => {
  if (!value) return texts.value.notAvailable;
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format("YYYY-MM-DD HH:mm:ss") : String(value);
};

const formatCoordinates = (x, y) => {
  if (x === undefined || x === null || y === undefined || y === null) {
    return texts.value.notAvailable;
  }
  return `${Number(x).toFixed(2)}, ${Number(y).toFixed(2)}`;
};

const copyText = async (value) => {
  if (!value) return;
  try {
    await navigator.clipboard.writeText(String(value));
    message.success(texts.value.copied);
  } catch (error) {
    message.error(texts.value.copyFail);
  }
};

watch(
  () => [linkStore.getCurrentUid(), linkStore.getUpdateStatus()],
  async ([uid, status]) => {
    if (!uid) return;
    if (status === "players") {
      activeTab.value = "players";
      await fetchPlayerDetail(uid);
      linkStore.setCurrentUid(null);
    }
  }
);

onMounted(async () => {
  await refreshAll(false);
  if (linkStore.getCurrentUid()) {
    await fetchPlayerDetail(linkStore.getCurrentUid(), true);
    linkStore.setCurrentUid(null);
  }
});
</script>

<template>
  <div class="min-h-screen bg-slate-50 dark:bg-slate-900">
    <div class="mx-auto max-w-7xl px-4 py-6">
      <n-space justify="space-between" align="center" class="mb-4" wrap>
        <div>
          <div class="text-2xl font-semibold">{{ texts.title }}</div>
          <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.subtitle }}</div>
        </div>
        <n-space>
          <n-button secondary @click="router.push('/ops')">{{ texts.backOps }}</n-button>
          <n-button secondary @click="router.push('/paldefender')">{{ texts.palDefenderWorkspace }}</n-button>
          <n-button secondary @click="router.push('/')">{{ texts.backHome }}</n-button>
        </n-space>
      </n-space>

      <n-alert v-if="!isLogin" type="info" class="mb-4">{{ texts.authNotice }}</n-alert>

      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="players" :tab="texts.playersTab">
          <n-card :title="texts.playerFilters" class="mb-4">
            <n-space vertical>
              <n-input v-model:value="filters.keyword" :placeholder="texts.keywordPlaceholder" @keyup.enter="fetchPlayers(false)" />
              <n-space wrap>
                <n-switch v-model:value="filters.online_only">
                  <template #checked>{{ texts.onlineOnly }}</template>
                  <template #unchecked>{{ texts.onlineOnly }}</template>
                </n-switch>
                <n-switch v-model:value="filters.whitelist_only">
                  <template #checked>{{ texts.whitelistOnly }}</template>
                  <template #unchecked>{{ texts.whitelistOnly }}</template>
                </n-switch>
                <n-switch v-model:value="filters.guild_only">
                  <template #checked>{{ texts.guildOnly }}</template>
                  <template #unchecked>{{ texts.guildOnly }}</template>
                </n-switch>
                <n-button type="primary" :loading="loading" @click="fetchPlayers(false)">{{ texts.refresh }}</n-button>
              </n-space>
            </n-space>
          </n-card>

          <n-grid cols="1 m:4" responsive="screen" x-gap="12" y-gap="12" class="mb-4">
            <n-gi><n-card><n-statistic :label="texts.totalPlayers" :value="stats.total" /></n-card></n-gi>
            <n-gi><n-card><n-statistic :label="texts.onlinePlayers" :value="stats.online" /></n-card></n-gi>
            <n-gi><n-card><n-statistic :label="texts.whitelistPlayers" :value="stats.whitelist" /></n-card></n-gi>
            <n-gi><n-card><n-statistic :label="texts.guildPlayers" :value="stats.guild" /></n-card></n-gi>
          </n-grid>

          <n-grid cols="1 l:2" responsive="screen" x-gap="12" y-gap="12">
            <n-gi>
              <n-card :title="texts.playerList">
                <n-space v-if="isLogin" vertical class="mb-4">
                  <n-space justify="space-between" align="center">
                    <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.selectedCount }}: {{ selectedPlayerUids.length }}</div>
                    <n-space>
                      <n-button size="small" secondary @click="toggleSelectVisible">{{ texts.selectVisible }}</n-button>
                      <n-button size="small" secondary @click="selectedPlayerUids = []">{{ texts.clearSelection }}</n-button>
                    </n-space>
                  </n-space>
                  <n-space wrap>
                    <n-button size="small" type="warning" :disabled="selectedPlayerUids.length === 0" :loading="batchLoading" @click="performBatchAction('whitelist_add')">{{ texts.addWhitelist }}</n-button>
                    <n-button size="small" type="default" :disabled="selectedPlayerUids.length === 0" :loading="batchLoading" @click="performBatchAction('whitelist_remove')">{{ texts.removeWhitelist }}</n-button>
                    <n-button size="small" type="error" :disabled="selectedPlayerUids.length === 0" :loading="batchLoading" @click="performBatchAction('ban')">{{ texts.ban }}</n-button>
                    <n-button size="small" type="warning" :disabled="selectedPlayerUids.length === 0" :loading="batchLoading" @click="performBatchAction('kick')">{{ texts.kick }}</n-button>
                    <n-button size="small" type="success" :disabled="selectedPlayerUids.length === 0" :loading="batchLoading" @click="performBatchAction('unban')">{{ texts.unban }}</n-button>
                  </n-space>
                </n-space>

                <n-spin :show="loading">
                  <n-empty v-if="players.length === 0" :description="texts.noData" />
                  <n-list v-else hoverable clickable>
                    <n-list-item
                      v-for="item in players"
                      :key="item.player_uid"
                      @click="fetchPlayerDetail(item.player_uid)"
                    >
                      <n-space vertical size="small" class="w-full">
                        <n-space justify="space-between" align="start" class="w-full">
                          <n-space align="center">
                            <n-checkbox
                              v-if="isLogin"
                              :checked="selectedUIDSet.has(item.player_uid)"
                              @update:checked="(checked) => togglePlayerSelection(item.player_uid, checked)"
                              @click.stop
                            />
                            <div class="font-medium">{{ item.nickname || item.player_uid }}</div>
                          </n-space>
                          <n-space>
                            <n-tag :type="item.online ? 'success' : 'default'" round>{{ item.online ? texts.online : texts.offline }}</n-tag>
                            <n-tag v-if="item.whitelisted" type="warning" round>{{ texts.whitelist }}</n-tag>
                            <n-tag v-if="item.guild" type="info" round>{{ item.guild.name }}</n-tag>
                          </n-space>
                        </n-space>
                        <div class="text-sm text-slate-500 dark:text-slate-300">UID: {{ item.player_uid }}</div>
                        <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.level }}: {{ item.level }} · Item {{ item.item_count }} · Pal {{ item.pal_count }}</div>
                        <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.lastOnline }}: {{ formatDateTime(item.last_online) }}</div>
                      </n-space>
                    </n-list-item>
                  </n-list>
                </n-spin>
              </n-card>
            </n-gi>

            <n-gi>
              <n-card :title="texts.playerDetail">
                <n-spin :show="detailLoading">
                  <n-empty v-if="!selectedSummary || !selectedPlayer" :description="texts.selectPlayer" />
                  <n-space v-else vertical>
                    <n-space justify="space-between" align="center">
                      <div class="text-lg font-semibold">{{ selectedSummary.nickname || selectedSummary.player_uid }}</div>
                      <n-space>
                        <n-tag :type="selectedSummary.online ? 'success' : 'default'" round>{{ selectedSummary.online ? texts.online : texts.offline }}</n-tag>
                        <n-tag v-if="selectedSummary.whitelisted" type="warning" round>{{ texts.whitelist }}</n-tag>
                        <n-tag v-if="selectedSummary.guild" type="info" round>{{ selectedSummary.guild.name }}</n-tag>
                      </n-space>
                    </n-space>

                    <n-card v-if="isLogin" size="small" :title="texts.playerActions">
                      <n-space wrap>
                        <n-button type="warning" secondary strong @click="performSingleAction(selectedSummary.whitelisted ? 'whitelist_remove' : 'whitelist_add')">
                          {{ selectedSummary.whitelisted ? texts.removeWhitelist : texts.addWhitelist }}
                        </n-button>
                        <n-button type="warning" secondary strong @click="performSingleAction('kick')">{{ texts.kick }}</n-button>
                        <n-button type="error" secondary strong @click="performSingleAction('ban')">{{ texts.ban }}</n-button>
                        <n-button type="success" secondary strong @click="performSingleAction('unban')">{{ texts.unban }}</n-button>
                      </n-space>
                    </n-card>

                    <n-card size="small" :title="texts.identifiers">
                      <n-space vertical size="small">
                        <div @click="copyText(selectedSummary.player_uid)">UID: {{ selectedSummary.player_uid }}</div>
                        <div @click="copyText(selectedSummary.user_id)">UserID: {{ selectedSummary.user_id || texts.notAvailable }}</div>
                        <div @click="copyText(selectedSummary.steam_id)">SteamID: {{ selectedSummary.steam_id || texts.notAvailable }}</div>
                        <div>{{ texts.accountName }}: {{ selectedSummary.account_name || texts.notAvailable }}</div>
                      </n-space>
                    </n-card>

                    <n-grid cols="1 m:2" responsive="screen" x-gap="12" y-gap="12">
                      <n-gi><n-card size="small"><n-statistic :label="texts.level" :value="selectedSummary.level" /></n-card></n-gi>
                      <n-gi><n-card size="small"><n-statistic :label="texts.buildingCount" :value="selectedSummary.building_count || 0" /></n-card></n-gi>
                      <n-gi><n-card size="small"><n-statistic :label="texts.lastOnline" :value="formatDateTime(selectedSummary.last_online)" /></n-card></n-gi>
                      <n-gi><n-card size="small"><n-statistic :label="texts.coordinates" :value="formatCoordinates(selectedSummary.location_x, selectedSummary.location_y)" /></n-card></n-gi>
                    </n-grid>

                    <n-card size="small" :title="texts.guild">
                      <n-empty v-if="!selectedSummary.guild" :description="texts.noData" />
                      <n-space v-else vertical size="small">
                        <div>{{ selectedSummary.guild.name }}</div>
                        <div>Admin UID: {{ selectedSummary.guild.admin_player_uid }}</div>
                        <div>BaseCamp Lv. {{ selectedSummary.guild.base_camp_level }} · Members {{ selectedSummary.guild.member_count }}</div>
                      </n-space>
                    </n-card>

                    <n-grid cols="1 m:2" responsive="screen" x-gap="12" y-gap="12">
                      <n-gi>
                        <n-card size="small" :title="texts.topItems">
                          <n-empty v-if="topItems.length === 0" :description="texts.noData" />
                          <n-space v-else vertical size="small">
                            <div v-for="item in topItems" :key="item.key">
                              {{ item.itemId }} × {{ item.stackCount }} · {{ item.container }}
                            </div>
                          </n-space>
                        </n-card>
                      </n-gi>
                      <n-gi>
                        <n-card size="small" :title="texts.palPreview">
                          <n-empty v-if="topPals.length === 0" :description="texts.noData" />
                          <n-space v-else vertical size="small">
                            <div v-for="pal in topPals" :key="`${pal.type}-${pal.nickname}-${pal.level}`">
                              {{ pal.nickname || pal.type }} · {{ texts.level }} {{ pal.level }}
                            </div>
                          </n-space>
                        </n-card>
                      </n-gi>
                    </n-grid>
                  </n-space>
                </n-spin>
              </n-card>
            </n-gi>
          </n-grid>
        </n-tab-pane>

        <n-tab-pane name="guilds" :tab="texts.guildsTab">
          <n-card>
            <n-space vertical>
              <n-input v-model:value="guildKeyword" :placeholder="texts.guildKeywordPlaceholder" />
              <n-spin :show="guildLoading">
                <n-empty v-if="filteredGuilds.length === 0" :description="texts.noData" />
                <n-grid v-else cols="1 l:2" responsive="screen" x-gap="12" y-gap="12">
                  <n-gi v-for="guild in filteredGuilds" :key="guild.admin_player_uid">
                    <n-card size="small">
                      <n-space vertical size="small">
                        <n-space justify="space-between" align="center">
                          <div class="font-medium">{{ guild.name }}</div>
                          <n-tag type="primary" round>Lv.{{ guild.base_camp_level }}</n-tag>
                        </n-space>
                        <div>Admin UID: {{ guild.admin_player_uid }}</div>
                        <div>{{ texts.memberCount }}: {{ (guild.players || []).length }}</div>
                        <div>{{ texts.baseCamp }}:</div>
                        <n-space vertical size="small">
                          <div v-for="camp in guild.base_camp || []" :key="camp.id">
                            {{ camp.id || 'basecamp' }} · {{ formatCoordinates(camp.location_x, camp.location_y) }} · R={{ camp.area }}
                          </div>
                        </n-space>
                        <n-divider style="margin: 4px 0" />
                        <n-space wrap>
                          <n-button
                            v-for="player in guild.players || []"
                            :key="player.player_uid"
                            size="small"
                            secondary
                            @click="openGuildPlayer(player.player_uid)"
                          >
                            {{ player.nickname || texts.viewPlayer }}
                          </n-button>
                        </n-space>
                      </n-space>
                    </n-card>
                  </n-gi>
                </n-grid>
              </n-spin>
            </n-space>
          </n-card>
        </n-tab-pane>

        <n-tab-pane name="items" :tab="texts.itemsTab">
          <n-card>
            <n-space vertical>
              <n-input v-model:value="itemKeyword" :placeholder="texts.searchKeyword" @keyup.enter="runItemSearch" />
              <n-space wrap>
                <n-switch v-model:value="searchSelectedPlayerOnly" :disabled="!selectedPlayerUid">
                  <template #checked>{{ texts.searchCurrentOnly }}</template>
                  <template #unchecked>{{ texts.searchCurrentOnly }}</template>
                </n-switch>
                <n-button type="primary" :disabled="!isLogin" :loading="itemSearching" @click="runItemSearch">{{ texts.searchItems }}</n-button>
              </n-space>
              <n-empty v-if="itemResults.length === 0" :description="texts.noData" />
              <n-list v-else bordered>
                <n-list-item v-for="item in itemResults" :key="`${item.player_uid}-${item.item_id}-${item.container}-${item.stack_count}`">
                  <n-space vertical size="small" class="w-full">
                    <n-space justify="space-between" align="center">
                      <div class="font-medium">{{ item.item_id }}</div>
                      <n-space>
                        <n-tag type="primary" round>{{ texts.itemCount }} {{ item.stack_count }}</n-tag>
                        <n-tag type="default" round>{{ item.container }}</n-tag>
                      </n-space>
                    </n-space>
                    <div>{{ texts.player }}: {{ item.nickname || item.player_uid }}</div>
                    <div>UID: {{ item.player_uid }} · Lv.{{ item.player_level }}</div>
                    <div v-if="item.guild_name">{{ texts.guild }}: {{ item.guild_name }}</div>
                  </n-space>
                </n-list-item>
              </n-list>
            </n-space>
          </n-card>
        </n-tab-pane>

        <n-tab-pane name="pals" :tab="texts.palsTab">
          <n-card>
            <n-space vertical>
              <n-input v-model:value="palKeyword" :placeholder="texts.searchKeyword" @keyup.enter="runPalSearch" />
              <n-space wrap>
                <n-switch v-model:value="searchSelectedPlayerOnly" :disabled="!selectedPlayerUid">
                  <template #checked>{{ texts.searchCurrentOnly }}</template>
                  <template #unchecked>{{ texts.searchCurrentOnly }}</template>
                </n-switch>
                <n-button type="primary" :disabled="!isLogin" :loading="palSearching" @click="runPalSearch">{{ texts.searchPals }}</n-button>
              </n-space>
              <n-empty v-if="palResults.length === 0" :description="texts.noData" />
              <n-list v-else bordered>
                <n-list-item v-for="pal in palResults" :key="`${pal.player_uid}-${pal.pal_id}-${pal.pal_nickname}-${pal.level}`">
                  <n-space vertical size="small" class="w-full">
                    <n-space justify="space-between" align="center">
                      <div class="font-medium">{{ pal.pal_nickname || pal.pal_id }}</div>
                      <n-space>
                        <n-tag type="primary" round>Lv.{{ pal.level }}</n-tag>
                        <n-tag v-if="pal.is_lucky" type="warning" round>Lucky</n-tag>
                        <n-tag v-if="pal.is_boss" type="error" round>Boss</n-tag>
                      </n-space>
                    </n-space>
                    <div>{{ texts.palId }}: {{ pal.pal_id }}</div>
                    <div>{{ texts.player }}: {{ pal.nickname || pal.player_uid }}</div>
                    <div v-if="pal.guild_name">{{ texts.guild }}: {{ pal.guild_name }}</div>
                    <div>{{ texts.palSkills }}: {{ (pal.skills || []).join(', ') || texts.noData }}</div>
                  </n-space>
                </n-list-item>
              </n-list>
            </n-space>
          </n-card>
        </n-tab-pane>

        <n-tab-pane name="map" :tab="texts.mapTab">
          <n-card>
            <n-space vertical>
              <div class="text-sm text-slate-500 dark:text-slate-300">{{ texts.mapHint }}</div>
              <div class="h-[70vh] overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
                <MapView />
              </div>
            </n-space>
          </n-card>
        </n-tab-pane>
      </n-tabs>
    </div>
  </div>
</template>
