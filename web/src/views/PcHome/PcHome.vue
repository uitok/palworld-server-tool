<script setup>
import {
  AdminPanelSettingsOutlined,
  SupervisedUserCircleRound,
  SettingsPowerRound,
  DeleteOutlineTwotone,
  RemoveRedEyeTwotone,
  DeleteFilled,
  ArchiveOutlined,
  CloudDownloadOutlined,
  PublicRound,
} from "@vicons/material";
import {
  GameController,
  LanguageSharp,
  ShieldCheckmarkSharp,
  Terminal,
  ArchiveOutline,
  Settings,
} from "@vicons/ionicons5";
import { GuiManagement } from "@vicons/carbon";
import { BroadcastTower } from "@vicons/fa";
import { computed, onMounted, ref } from "vue";
import { NTag, NButton, NIcon, useMessage, useDialog } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import pageStore from "@/stores/model/page.js";
import dayjs from "dayjs";
import palMap from "@/assets/pal.json";
import itemMap from "@/assets/items.json";
import skillMap from "@/assets/skill.json";
import PlayerList from "./component/PlayerList.vue";
import GuildList from "./component/GuildList.vue";
import MapView from "./component/MapView.vue";
import whitelistStore from "@/stores/model/whitelist";
import playerToGuildStore from "@/stores/model/playerToGuild";
import { watch } from "vue";
import userStore from "@/stores/model/user";
import { h } from "vue";
import PalDefenderBatchOperations from "@/components/PalDefenderBatchOperations.vue";
import { explainPalDefenderError, isPlayerOnlineForLiveGrant } from "@/utils/paldefender";

const { t, locale } = useI18n();

const message = useMessage();
const dialog = useDialog();

const PALWORLD_TOKEN = "palworld_token";
const SUPPORTED_LOCALES = ["zh", "en", "ja"];
const getSafeLocale = () => {
  const savedLocale = localStorage.getItem("locale");
  return SUPPORTED_LOCALES.includes(savedLocale) ? savedLocale : "zh";
};

const pageWidth = computed(() => pageStore().getScreenWidth());
const smallScreen = computed(() => pageWidth.value < 1024);

const loading = ref(false);
const serverInfo = ref({});
const serverMetrics = ref({});
const currentDisplay = ref("players");
const playerList = ref([]);
const onlinePlayerList = ref([]);
const guildList = ref([]);
const playerInfo = ref({});
const playerPalsList = ref([]);
const guildInfo = ref({});
const skillTypeList = ref([]);
const languageOptions = ref([]);

const isLogin = ref(false);
const authToken = ref("");

const isDarkMode = ref(
  window.matchMedia("(prefers-color-scheme: dark)").matches
);

const updateDarkMode = (e) => {
  isDarkMode.value = e.matches;
};

const getDarkModeColor = () => {
  return isDarkMode.value ? "#fff" : "#000";
};

const getUserAvatar = () => {
  return new URL("../../assets/avatar.webp", import.meta.url).href;
};

const handleSelectLanguage = (key) => {
  message.info(t("message.changelanguage"));
  if (key === "zh") {
    localStorage.setItem("locale", "zh");
    // locale.value = "zh";
  } else if (key === "ja") {
    localStorage.setItem("locale", "ja");
    // locale.value = "ja";
  } else {
    localStorage.setItem("locale", "en");
    // locale.value = "en";
  }
  setTimeout(() => {
    location.reload();
  }, 1000);
};

const getSkillTypeList = () => {
  const currentSkillMap = skillMap[locale.value] || skillMap.zh || {};
  return Object.values(currentSkillMap).map((item) => item.name);
};

const toPalConf = () => {
  window.open("/pal-conf");
};

const toGithub = () => {
  window.open("https://github.com/zaigie/palworld-server-tool/releases");
};
const serverToolInfo = ref({});
const hasNewVersion = ref(false);
const getServerToolInfo = async () => {
  const { data } = await new ApiService().getServerToolInfo();
  serverToolInfo.value = data.value;
  if (data.value) {
    hasNewVersion.value = isNewVersion(data.value?.version, data.value?.latest);
  }
};
const isNewVersion = (version, latest) => {
  if (version == "Unknown" || version == "Develop" || latest == "") {
    return false;
  }
  const currentVersion = version.split("v")[1];
  const latestVersion = latest?.split("v")[1];
  const currentParts = currentVersion.substring(1).split(".");
  const latestParts = latestVersion.substring(1).split(".");
  for (let i = 0; i < currentParts.length; i++) {
    const currentPart = parseInt(currentParts[i], 10);
    const latestPart = parseInt(latestParts[i], 10);
    if (latestPart > currentPart) {
      return true;
    } else if (latestPart < currentPart) {
      return false;
    }
  }
  return false;
};

// get data
const getServerInfo = async () => {
  const { data, statusCode } = await new ApiService().getServerInfo();
  if (statusCode.value === 200 && data.value) {
    serverInfo.value = data.value;
  } else {
    serverInfo.value = {
      name: "REST API unavailable",
      version: "",
    };
  }
};

const getServerMetrics = async () => {
  const { data, statusCode } = await new ApiService().getServerMetrics();
  if (statusCode.value === 200 && data.value) {
    serverMetrics.value = data.value;
  } else {
    serverMetrics.value = {
      current_player_num: 0,
    };
  }
};

const getPlayerList = async () => {
  getOnlineList();
  const { data } = await new ApiService().getPlayerList({
    order_by: "last_online",
    desc: true,
  });
  playerList.value = data.value;
  rconPlayerOptions.value = data.value.map((item) => {
    const online = dayjs().diff(dayjs(item.last_online)) < 80000;
    return {
      label: `[${online ? t("status.online") : t("status.offline")}] ${item.nickname}(${item.player_uid})`,
      value: `${item.player_uid}|${item.user_id}|${item.steam_id}|${item.last_online || ""}`,
    };
  });
};
const getOnlineList = async () => {
  const { data } = await new ApiService().getOnlinePlayerList();
  onlinePlayerList.value = data.value;
};

// login
const showLoginModal = ref(false);
const password = ref("");
const handleLogin = async () => {
  const { data, statusCode } = await new ApiService().login({
    password: password.value,
  });
  if (statusCode.value === 401) {
    message.error(t("message.autherr"));
    password.value = "";
    return;
  }
  let token = data.value.token;
  localStorage.setItem(PALWORLD_TOKEN, token);
  userStore().setIsLogin(true, token);
  await getWhiteList();
  authToken.value = token;
  message.success(t("message.authsuccess"));
  showLoginModal.value = false;
  isLogin.value = true;
};
const showRconDrawer = ref(false);
const rconCommands = ref([]);
const rconSelectedPlayer = ref(null);
const rconPlayerOptions = ref([]);
const rconSelectedItem = ref(null);
const rconItemOptions = ref([]);
const rconSelectedPal = ref(null);
const rconPalOptions = ref([]);
const rconSelectedEgg = ref(null);
const rconEggOptions = ref([]);
const rconCommandAmount = ref(1);
const rconCommandLevel = ref(1);
const rconCommandsExtra = ref({});
const copyText = async (text) => {
  if (text == "" || text == null) {
    message.error(t("message.copyempty"));
    return;
  }
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      message.success(t("message.copysuccess"));
    } catch (err) {
      message.error(t("message.copyerr", { err }));
    }
  } else {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
      message.success(t("message.copysuccess"));
    } catch (err) {
      message.error(t("message.copyerr", { err }));
    }
    document.body.removeChild(textarea);
  }
};
const handleRconDrawer = () => {
  if (checkAuthToken()) {
    showRconDrawer.value = true;
    getRconCommands();
  } else {
    message.error(t("message.requireauth"));
    showRconDrawer.value = true;
  }
};
const getRconCommands = async () => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().getRconCommands();
    if (statusCode.value === 200) {
      rconCommands.value = data.value;
      rconCommands.value.forEach((item) => {
        rconCommandsExtra.value[item.uuid] = "";
      });
    }
  }
};
const sendRconCommand = async (uuid) => {
  const content = rconCommandsExtra.value[uuid];
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().sendRconCommand({
      uuid,
      content,
    });
    if (statusCode.value === 200) {
      message.info(data.value?.message);
    } else {
      message.error(t("message.rconfail", { err: data.value?.error }));
    }
  }
};
const sendRawRconCommand = async (command) => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().sendRawRconCommand({
      command,
    });
    if (statusCode.value === 200) {
      message.info(data.value?.message || t("message.rconsuccess"));
      return true;
    }
    message.error(t("message.rconfail", { err: data.value?.error }));
  }
  return false;
};
const getSelectedRconPlayerParts = () => {
  return rconSelectedPlayer.value ? rconSelectedPlayer.value.split("|") : [];
};
const getSelectedPlayerPayload = () => {
  const playerParts = getSelectedRconPlayerParts();
  if (!playerParts.length) {
    message.warning(t("message.selectPlayerFirst"));
    return null;
  }
  const [playerUid, userId, steamId, lastOnline] = playerParts;
  return {
    playerUid,
    player_uid: playerUid,
    user_id: userId || "",
    steam_id: steamId || "",
    last_online: lastOnline || "",
  };
};
const getSelectedSteamUserId = () => {
  const playerPayload = getSelectedPlayerPayload();
  if (!playerPayload) {
    return null;
  }
  if (!playerPayload.steam_id) {
    message.warning(t("message.selectPlayerFirst"));
    return null;
  }
  return "steam_" + playerPayload.steam_id;
};
const ensureSelectedLiveGrantReady = () => {
  if (!checkAuthToken()) {
    message.error(t("message.requireauth"));
    return null;
  }
  const playerPayload = getSelectedPlayerPayload();
  if (!playerPayload) {
    return null;
  }
  if (!isPlayerOnlineForLiveGrant(playerPayload.last_online)) {
    message.warning(t("message.playerMustBeOnline"));
    return null;
  }
  return playerPayload;
};
const getPositiveNumber = (value, fallback = 1) => {
  const parsed = Number.parseInt(String(value ?? fallback), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
};
const quickGiveItem = async () => {
  const playerPayload = ensureSelectedLiveGrantReady();
  if (!playerPayload) {
    return;
  }
  if (!rconSelectedItem.value) {
    message.warning(t("message.selectItemFirst"));
    return;
  }
  const { data, statusCode } = await new ApiService().grantPlayerItems({
    ...playerPayload,
    item_id: rconSelectedItem.value,
    amount: getPositiveNumber(rconCommandAmount.value),
  });
  if (statusCode.value === 200) {
    message.success(t("message.itemActionSuccess"));
    return;
  }
  message.error(t("message.itemActionFail", { err: explainPalDefenderError(t, data.value) }));
};
const quickGivePal = async () => {
  const playerPayload = ensureSelectedLiveGrantReady();
  if (!playerPayload) {
    return;
  }
  if (!rconSelectedPal.value) {
    message.warning(t("message.selectPalFirst"));
    return;
  }
  const { data, statusCode } = await new ApiService().grantPlayerPal({
    ...playerPayload,
    pal_id: rconSelectedPal.value,
    level: getPositiveNumber(rconCommandLevel.value),
    amount: getPositiveNumber(rconCommandAmount.value),
  });
  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(t("message.palActionFail", { err: explainPalDefenderError(t, data.value) }));
};
const quickGiveEgg = async () => {
  const playerPayload = ensureSelectedLiveGrantReady();
  if (!playerPayload) {
    return;
  }
  if (!rconSelectedEgg.value) {
    message.warning(t("message.selectEggFirst"));
    return;
  }
  if (!rconSelectedPal.value) {
    message.warning(t("message.selectPalFirst"));
    return;
  }
  const { data, statusCode } = await new ApiService().grantPlayerPalEgg({
    ...playerPayload,
    egg_id: rconSelectedEgg.value,
    pal_id: rconSelectedPal.value,
    level: getPositiveNumber(rconCommandLevel.value),
    amount: getPositiveNumber(rconCommandAmount.value),
  });
  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(t("message.palActionFail", { err: explainPalDefenderError(t, data.value) }));
};
const quickGrantByAmount = async (kind) => {
  const playerPayload = ensureSelectedLiveGrantReady();
  if (!playerPayload) {
    return;
  }
  const { data, statusCode } = await new ApiService().grantPlayerSupport({
    ...playerPayload,
    kind,
    amount: getPositiveNumber(rconCommandAmount.value),
  });
  if (statusCode.value === 200) {
    message.success(t("message.palActionSuccess"));
    return;
  }
  message.error(t("message.palActionFail", { err: explainPalDefenderError(t, data.value) }));
};
const buildRconSelectionOptions = () => {
  const currentItems = itemMap[locale.value] || itemMap.zh || [];
  rconItemOptions.value = currentItems.map((item) => {
    return {
      label: `${item.name}-${item.key}`,
      value: item.key,
    };
  });
  rconEggOptions.value = currentItems
    .filter((item) => /^PalEgg_/i.test(item.key))
    .map((item) => {
      return {
        label: `${item.name}-${item.key}`,
        value: item.key,
      };
    });
  rconPalOptions.value = Object.entries(palMap[locale.value] || palMap.zh || {}).map(
    ([key, value]) => {
      return {
        label: `${value}-${key}`,
        value: key,
      };
    }
  );
};

const showRconAddModal = ref(false);
const newRconCommand = ref("");
const newRconPlaceholder = ref("");
const newRconRemark = ref("");
const handleAddRconCommand = () => {
  showRconAddModal.value = true;
  newRconCommand.value = "";
  newRconPlaceholder.value = "";
  newRconRemark.value = "";
};
const handleImportRconFinish = (options) => {
  getRconCommands();
  setTimeout(() => {
    message.success(t("message.importRconSuccess"));
    showRconAddModal.value = false;
  }, 500);
};
const handleImportRconError = (options) => {
  let err = options.event?.target?.response
    ? JSON.parse(options.event?.target?.response).error
    : "";
  message.error(t("message.importRconFail", { err }));
};
const importRconPreset = async (name) => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().importRconPreset({ name });
    if (statusCode.value === 200) {
      message.success(t("message.importRconPresetSuccess"));
      await getRconCommands();
      return;
    }
    message.error(
      t("message.importRconPresetFail", { err: data.value?.error || "" })
    );
  }
};
const addRconCommand = async () => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().addRconCommand({
      command: newRconCommand.value,
      placeholder: newRconPlaceholder.value,
      remark: newRconRemark.value,
    });
    if (statusCode.value === 200) {
      message.success(t("message.addrconsuccess"));
      await getRconCommands();
      newRconCommand.value = "";
      newRconPlaceholder.value = "";
      newRconRemark.value = "";
    } else {
      message.error(t("message.addrconfail", { err: data.value?.error }));
    }
  }
};
const removeRconCommand = async (uuid) => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().removeRconCommand(uuid);
    if (statusCode.value === 200) {
      message.success(t("message.removerconsuccess"));
      await getRconCommands();
    } else {
      message.error(t("message.removerconfail", { err: data.value?.error }));
    }
  }
};
const fillRconCommand = (rconCommand) => {
  let cmd = rconCommand.placeholder;
  const playerParts = getSelectedRconPlayerParts();
  if (/{steamUserID}/i.test(cmd)) {
    if (!playerParts.length) {
      message.warning(t("message.selectPlayerFirst"));
      return;
    }
    cmd = cmd.replace(/{steamUserID}/gi, "steam_" + playerParts[2]);
  }
  if (/{userID}/i.test(cmd)) {
    if (!playerParts.length) {
      message.warning(t("message.selectPlayerFirst"));
      return;
    }
    cmd = cmd.replace(/{userID}/gi, playerParts[1]);
  }
  if (/{itemID}/i.test(cmd)) {
    if (!rconSelectedItem.value) {
      message.warning(t("message.selectItemFirst"));
      return;
    }
    cmd = cmd.replace(/{itemID}/gi, rconSelectedItem.value);
  }
  if (/{palID}/i.test(cmd)) {
    if (!rconSelectedPal.value) {
      message.warning(t("message.selectPalFirst"));
      return;
    }
    cmd = cmd.replace(/{palID}/gi, rconSelectedPal.value);
  }
  if (/{eggID}/i.test(cmd) || /{帕鲁蛋ID}/.test(cmd)) {
    if (!rconSelectedEgg.value) {
      message.warning(t("message.selectEggFirst"));
      return;
    }
    cmd = cmd.replace(/{eggID}/gi, rconSelectedEgg.value).replace(/{帕鲁蛋ID}/g, rconSelectedEgg.value);
  }
  cmd = cmd
    .replace(/{amount}/gi, String(getPositiveNumber(rconCommandAmount.value)))
    .replace(/{数量}/g, String(getPositiveNumber(rconCommandAmount.value)))
    .replace(/{level}/gi, String(getPositiveNumber(rconCommandLevel.value)))
    .replace(/{等级}/g, String(getPositiveNumber(rconCommandLevel.value)));
  rconCommandsExtra.value[rconCommand.uuid] = cmd;
};

// 控制中心（下拉菜单）
// 包含：白名单管理、RCON 命令、游戏内广播、关闭服务器
const renderIcon = (icon, color = "#666") => {
  return () => {
    return h(
      NIcon,
      {
        color: color,
      },
      {
        default: () => h(icon),
      }
    );
  };
};
const controlCenterOption = [
  // {
  //   label: () => {
  //     return h("div", null, {
  //       default: () => t("button.backup"),
  //     });
  //   },
  //   key: "backup",
  //   icon: renderIcon(ArchiveOutlined),
  // },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.palconf"),
      });
    },
    key: "palconf",
    icon: renderIcon(Settings),
  },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.whitelist"),
      });
    },
    key: "whitelist",
    icon: renderIcon(ShieldCheckmarkSharp),
  },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.rcon"),
      });
    },
    key: "rcon",
    icon: renderIcon(Terminal),
  },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.broadcast"),
      });
    },
    key: "broadcast",
    icon: renderIcon(BroadcastTower),
  },
  {
    label: () => {
      return h(
        "div",
        {
          style: { color: "#cc2d48" },
        },
        {
          default: () => t("button.shutdown"),
        }
      );
    },
    key: "shutdown",
    icon: renderIcon(SettingsPowerRound, "#cc2d48"),
  },
];
const handleSelectControlCenter = (key) => {
  if (key === "palconf") {
    toPalConf();
  } else if (key === "whitelist") {
    handleWhiteList();
  } else if (key === "rcon") {
    handleRconDrawer();
  } else if (key === "broadcast") {
    handleStartBrodcast();
  } else if (key === "shutdown") {
    handleShutdown();
  } else {
    message.error("错误");
  }
};

// 白名单
const showWhiteListModal = ref(false);
const whiteList = ref([]);
const handleWhiteList = () => {
  if (checkAuthToken()) {
    showWhiteListModal.value = true;
    getWhiteList();
  } else {
    message.error(t("message.requireauth"));
    showWhiteListModal.value = true;
  }
};
const getWhiteList = async () => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().getWhitelist();
    if (statusCode.value === 200) {
      if (data.value) {
        whitelistStore().setWhitelist(data.value);
        whiteList.value = [];
        data.value.forEach((item) => {
          whiteList.value.push({
            ...item,
            isNew: false,
          });
        });
      } else {
        whiteList.value = [];
      }
    }
  }
};
// 查看白名单中的该玩家
const showWhitelistPlayer = ref(null);
const showCurrentPlayer = (id) => {
  showWhitelistPlayer.value = id;
  showWhiteListModal.value = false;
};
// 从白名单中移除该玩家
const removeWhiteList = async (player) => {
  if (!player.player_uid && !player.steam_id) {
    message.error(
      t("message.removewhitefail", {
        err: "player_uid or steam_id is required",
      })
    );
    return;
  }
  if (player.isNew) {
    const index = whiteList.value.findIndex(
      (e) => e.player_uid === player.player_uid
    );
    whiteList.value.splice(index, 1);
  } else {
    const { data, statusCode } = await new ApiService().removeWhitelist(player);
    if (statusCode.value === 200) {
      message.success(t("message.removewhitesuccess"));
      await getWhiteList();
    } else {
      message.error(t("message.removewhitefail", { err: data.value?.error }));
    }
  }
};
// 添加一项到白名单中
const virtualListInst = ref();
const handleAddNewWhiteList = () => {
  whiteList.value.unshift({
    name: "",
    player_uid: "",
    steam_id: "",
    isNew: true,
  });
  virtualListInst.value?.scrollTo({ index: 0 });
};
// 保存修改白名单
const putWhiteList = async () => {
  if (whiteList.value.length === 0) {
    return;
  }
  const whiteListData = JSON.stringify(whiteList.value);
  const { data, statusCode } = await new ApiService().putWhitelist(
    whiteListData
  );
  if (statusCode.value === 200) {
    message.success(t("message.addwhitesuccess"));
    getWhiteList();
    showWhiteListModal.value = false;
  } else {
    message.error(t("message.addwhitefail", { err: data.value?.error }));
  }
};
// 接受玩家加入到黑名单信息
const getSonWhitelistStatus = () => {
  getWhiteList();
};

// 广播
const showBroadcastModal = ref(false);
const broadcastText = ref("");
const handleStartBrodcast = () => {
  // 开始广播
  if (checkAuthToken()) {
    showBroadcastModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};
const handleBroadcast = async () => {
  const { data, statusCode } = await new ApiService().sendBroadcast({
    message: broadcastText.value,
  });
  if (statusCode.value === 200) {
    message.success(t("message.broadcastsuccess"));
    showBroadcastModal.value = false;
    broadcastText.value = "";
  } else {
    message.error(t("message.broadcastfail", { err: data.value?.error }));
  }
};

const doShutdown = async () => {
  return await new ApiService().shutdownServer({
    seconds: 60,
    message: "Server Will Shutdown After 60 Seconds",
  });
};

// 关机
const handleShutdown = () => {
  if (checkAuthToken()) {
    dialog.warning({
      title: t("message.warn"),
      content: t("message.shutdowntip"),
      positiveText: t("button.confirm"),
      negativeText: t("button.cancel"),
      onPositiveClick: async () => {
        const { data, statusCode } = await doShutdown();
        if (statusCode.value === 200) {
          message.success(t("message.shutdownsuccess"));
          return;
        } else {
          message.error(t("message.shutdownfail", { err: data.value?.error }));
        }
      },
      onNegativeClick: () => {},
    });
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};

const toPlayers = async () => {
  if (currentDisplay.value === "players") {
    return;
  }
  currentDisplay.value = "players";
  playerToGuildStore().setUpdateStatus("players");
};
const toGuilds = async () => {
  if (currentDisplay.value === "guilds") {
    return;
  }
  currentDisplay.value = "guilds";
  playerToGuildStore().setUpdateStatus("guilds");
};

const toMap = async () => {
  if (currentDisplay.value === "map") {
    return;
  }
  currentDisplay.value = "map";
  playerToGuildStore().setUpdateStatus("map");
};

const playerToGuildStatus = computed(() =>
  playerToGuildStore().getUpdateStatus()
);

watch(
  () => playerToGuildStatus.value,
  (newVal) => {
    currentDisplay.value = newVal;
    if (newVal === "players") {
    } else if (newVal === "guilds") {
    }
  }
);

/**
 * 检测 token
 */
const checkAuthToken = () => {
  const token = localStorage.getItem(PALWORLD_TOKEN);
  if (token && token !== "") {
    if (isTokenExpired(token)) {
      localStorage.removeItem(PALWORLD_TOKEN);
      return false;
    }
    isLogin.value = true;
    authToken.value = token;
    return true;
  }
  return false;
};
const isTokenExpired = (token) => {
  const payload = JSON.parse(atob(token.split(".")[1]));
  return payload.exp < Date.now() / 1000;
};

const backupModal = ref(false);
const backupList = ref([]);

const handleBackupList = () => {
  if (checkAuthToken()) {
    backupModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};
const getBackupList = async () => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().getBackupList({
      startTime: range.value[0],
      endTime: range.value[1],
    });
    if (statusCode.value === 200) {
      backupList.value = data.value;
    }
  }
};
const getBackupListWithRange = async (selectRange) => {
  let startTime = selectRange[0] ? selectRange[0] : 0;
  let endTime = selectRange[1] ? selectRange[1] : 0;
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().getBackupList({
      startTime,
      endTime,
    });
    if (statusCode.value === 200) {
      backupList.value = data.value;
    }
  }
};
const backupColumns = [
  {
    title: t("item.time"),
    key: "save_time",
    width: "200px",
    render: (row) => {
      return dayjs(row.save_time).format("YYYY-MM-DD HH:mm:ss");
    },
  },
  // {
  //   title: t("item.backupFile"),
  //   key: "path",
  //   render: (row) => {
  //     return row.path;
  //   },
  // },
  {
    title: "",
    key: "action",
    width: "200px",
    render: (row) => {
      return [
        h(
          NButton,
          {
            type: "primary",
            size: "small",
            renderIcon: () => h(CloudDownloadOutlined),
            onClick: () => downloadBackup(row),
          },
          { default: () => t("button.download") }
        ),
        h(
          NButton,
          {
            type: "error",
            size: "small",
            renderIcon: () => h(DeleteOutlineTwotone),
            style: "margin-left: 20px",
            onClick: () => removeBackup(row),
          },
          { default: () => t("button.remove") }
        ),
      ];
    },
  },
];

const range = ref([Date.now() - 1 * 24 * 60 * 60 * 1000, Date.now()]);
const isDownloading = ref(false);
const removeBackup = async (item) => {
  if (checkAuthToken()) {
    isDownloading.value = true;
    const { data, statusCode } = await new ApiService().removeBackup(
      item.backup_id
    );
    if (statusCode.value === 200) {
      message.success(t("message.removebackupsuccess"));
      await getBackupList();
    } else {
      message.error(t("message.removebackupfail", { err: data.value?.error }));
    }
    isDownloading.value = false;
  }
};

const downloadBackup = async (item) => {
  if (checkAuthToken()) {
    isDownloading.value = true;
    try {
      const { data: blobData, execute: fetchBlob } =
        await new ApiService().downloadBackup(item.backup_id);
      await fetchBlob();
      const url = URL.createObjectURL(blobData.value);
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", item.path);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      message.success(t("message.downloadsuccess"));
    } catch (error) {
      console.error("Download failed", error);
    }
    isDownloading.value = false;
  }
};

onMounted(async () => {
  locale.value = getSafeLocale();
  localStorage.setItem("locale", locale.value);
  languageOptions.value = [
    {
      label: "简体中文",
      key: "zh",
      disabled: locale.value == "zh",
    },
    {
      label: "English",
      key: "en",
      disabled: locale.value == "en",
    },
    {
      label: "日本語",
      key: "ja",
      disabled: locale.value == "ja",
    },
  ];
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  mediaQuery.addEventListener("change", updateDarkMode);
  isDarkMode.value = mediaQuery.matches;

  buildRconSelectionOptions();
  skillTypeList.value = getSkillTypeList();
  checkAuthToken();
  getServerInfo();
  getServerMetrics();
  getServerToolInfo();
  getPlayerList();
  await getWhiteList();
  await getBackupList();
  setInterval(async () => {
    await getPlayerList();
    await getServerMetrics();
  }, 60000);
  // 调试用
  // currentDisplay.value = "map";
  // playerToGuildStore().setUpdateStatus("map");
});
</script>

<template>
  <div class="home-page">
    <div
      :class="isDarkMode ? 'bg-#18181c text-#fff' : 'bg-#fff text-#18181c'"
      class="flex justify-between items-center p-3"
    >
      <n-space class="flex items-center">
        <span
          class="line-clamp-1"
          :class="smallScreen ? 'text-lg' : 'text-2xl'"
          >{{ $t("title") }}</span
        >
        <n-badge
          v-if="serverToolInfo?.version"
          :value="hasNewVersion ? 'new' : ''"
        >
          <n-tag
            type="warning"
            :size="smallScreen ? 'mini' : 'medium'"
            round
            @click="toGithub"
            style="cursor: pointer"
            >{{ serverToolInfo.version }}</n-tag
          >
        </n-badge>
        <n-tooltip trigger="hover">
          <template #trigger>
            <n-tag type="default" :size="smallScreen ? 'medium' : 'large'">{{
              serverInfo?.name
                ? `${serverInfo.name + " " + serverInfo.version}`
                : $t("message.loading")
            }}</n-tag>
          </template>
          <div>
            <p>{{ $t("item.serverFps") }}: {{ serverMetrics?.server_fps }}</p>
            <p>{{ $t("item.serverUptime") }}: {{ serverMetrics?.uptime }}(s)</p>
            <p>{{ $t("item.serverDays") }}: {{ serverMetrics?.days }}</p>
            <p>
              {{ $t("item.serverFrameTime") }}:
              {{ serverMetrics?.server_frame_time }}(ms)
            </p>
            <p>
              {{ $t("item.maxPlayerNum") }}: {{ serverMetrics?.max_player_num }}
            </p>
          </div>
        </n-tooltip>
      </n-space>

      <n-space>
        <n-dropdown
          trigger="hover"
          :options="languageOptions"
          @select="handleSelectLanguage"
        >
          <n-button type="default" secondary strong circle>
            <template #icon>
              <n-icon><LanguageSharp /></n-icon>
            </template>
          </n-button>
        </n-dropdown>

        <n-button
          type="primary"
          secondary
          strong
          @click="showLoginModal = true"
          v-if="!isLogin"
        >
          <template #icon>
            <n-icon>
              <AdminPanelSettingsOutlined />
            </n-icon>
          </template>
          {{ $t("button.auth") }}
        </n-button>
        <n-tag v-else type="success" size="large" round>
          <template #icon>
            <n-icon>
              <AdminPanelSettingsOutlined />
            </n-icon>
          </template>
          {{ $t("status.authenticated") }}
        </n-tag>
      </n-space>
    </div>
    <div class="w-full">
      <div class="rounded-lg" v-if="!loading">
        <n-layout style="height: calc(100vh - 64px)">
          <n-layout-header class="p-3 flex justify-between h-16" bordered>
            <n-button-group :size="smallScreen ? 'medium' : 'large'">
              <n-button
                @click="toPlayers"
                :type="currentDisplay === 'players' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <GameController />
                  </n-icon>
                </template>
                {{ $t("button.players") }}
              </n-button>
              <n-button
                @click="toGuilds()"
                :type="currentDisplay === 'guilds' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <SupervisedUserCircleRound />
                  </n-icon>
                </template>
                {{ $t("button.guilds") }}
              </n-button>
              <n-button
                @click="toMap()"
                :type="currentDisplay === 'map' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <PublicRound />
                  </n-icon>
                </template>
                {{ $t("button.map") }}
              </n-button>
            </n-button-group>
            <n-space>
              <n-tag type="info" round size="large">{{
                $t("status.player_number", { number: playerList?.length })
              }}</n-tag>
              <n-tag type="success" round size="large">{{
                $t("status.online_number", {
                  number: serverMetrics?.current_player_num,
                })
              }}</n-tag>
            </n-space>
            <n-space v-if="isLogin" class="flex items-center">
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="success"
                secondary
                strong
                round
                @click="handleBackupList"
              >
                <template #icon>
                  <n-icon>
                    <ArchiveOutlined />
                  </n-icon>
                </template>
                {{ $t("button.backup") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="primary"
                secondary
                strong
                round
                @click="handleRconDrawer"
              >
                <template #icon>
                  <n-icon>
                    <Terminal />
                  </n-icon>
                </template>
                {{ $t("button.rcon") }}
              </n-button>
              <n-dropdown
                trigger="click"
                size="large"
                :options="controlCenterOption"
                @select="handleSelectControlCenter"
              >
                <n-button
                  :size="smallScreen ? 'medium' : 'large'"
                  type="error"
                  secondary
                  strong
                  round
                >
                  <template #icon>
                    <n-icon>
                      <GuiManagement />
                    </n-icon>
                  </template>
                  {{ $t("button.controlCenter") }}</n-button
                >
              </n-dropdown>
              <!-- <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="default"
                secondary
                strong
                round
                @click="toPalConf"
              >
                <template #icon>
                  <n-icon>
                    <Settings />
                  </n-icon>
                </template>
                {{ $t("button.palconf") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="warning"
                secondary
                strong
                round
                @click="handleWhiteList"
              >
                <template #icon>
                  <n-icon>
                    <ShieldCheckmarkSharp />
                  </n-icon>
                </template>
                {{ $t("button.whitelist") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="success"
                secondary
                strong
                round
                @click="handleStartBrodcast"
              >
                <template #icon>
                  <n-icon>
                    <BroadcastTower />
                  </n-icon>
                </template>
                {{ $t("button.broadcast") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="error"
                secondary
                strong
                round
                @click="handleShutdown"
              >
                <template #icon>
                  <n-icon>
                    <SettingsPowerRound />
                  </n-icon>
                </template>
                {{ $t("button.shutdown") }}
              </n-button> -->
            </n-space>
          </n-layout-header>
          <div class="overflow-hidden" style="height: calc(100% - 64px)">
            <player-list
              v-if="currentDisplay === 'players'"
              :showWhitelistPlayer="showWhitelistPlayer"
              @onWhitelistStatus="getSonWhitelistStatus"
            ></player-list>
            <guild-list v-if="currentDisplay === 'guilds'"></guild-list>
            <map-view v-if="currentDisplay === 'map'"></map-view>
          </div>
        </n-layout>
      </div>
    </div>
  </div>
  <!-- login modal -->
  <n-modal
    v-model:show="showLoginModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 600px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.auth')"
    size="huge"
    :bordered="false"
    :segmented="segmented"
  >
    <div>
      <span class="block pb-2">{{ $t("message.authdesc") }}</span>
      <n-input
        type="password"
        show-password-on="click"
        size="large"
        v-model:value="password"
      ></n-input>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <n-button
          type="tertiary"
          @click="
            () => {
              showLoginModal = false;
              password = '';
            }
          "
          >{{ $t("button.cancel") }}</n-button
        >
        <n-button class="ml-3 w-40" type="primary" @click="handleLogin">{{
          $t("button.confirm")
        }}</n-button>
      </div>
    </template>
  </n-modal>

  <!-- broadcast modal -->
  <n-modal
    v-model:show="showBroadcastModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 600px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.broadcast')"
    size="huge"
    :bordered="false"
    :segmented="segmented"
  >
    <div>
      <n-input
        type="text"
        show-password-on="click"
        v-model:value="broadcastText"
      ></n-input>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <n-button
          type="tertiary"
          @click="
            () => {
              showBroadcastModal = false;
              broadcastText = '';
            }
          "
          >{{ $t("button.cancel") }}</n-button
        >
        <n-button class="ml-3 w-40" type="primary" @click="handleBroadcast">{{
          $t("button.confirm")
        }}</n-button>
      </div>
    </template>
  </n-modal>

  <!-- custom rcon drawer -->
  <n-modal
    v-model:show="showRconAddModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 600px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('button.addRcon')"
    size="huge"
    :bordered="false"
    :segmented="segmented"
  >
    <n-tabs default-value="preset" size="large" justify-content="space-evenly">
      <n-tab-pane name="preset" :tab="$t('button.presetPack')">
        <n-space vertical>
          <n-alert type="info" :show-icon="false">
            {{ $t("message.rconPresetOfficialDesc") }}
          </n-alert>
          <n-button type="primary" strong secondary @click="importRconPreset('official')">
            {{ $t("button.importOfficialPreset") }}
          </n-button>
          <n-alert type="warning" :show-icon="false">
            {{ $t("message.rconPresetPalDefenderDesc") }}
          </n-alert>
          <n-button type="warning" strong secondary @click="importRconPreset('paldefender')">
            {{ $t("button.importPalDefenderPreset") }}
          </n-button>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="import" :tab="$t('button.import')">
        <n-upload
          multiple
          directory-dnd
          action="/api/rcon/import"
          :headers="{ Authorization: `Bearer ${authToken}` }"
          :max="1"
          @finish="handleImportRconFinish"
          @error="handleImportRconError"
        >
          <n-upload-dragger>
            <div style="margin-bottom: 12px">
              <n-icon size="48" :depth="3">
                <ArchiveOutline />
              </n-icon>
            </div>
            <n-text style="font-size: 16px">
              {{ $t("message.importRconTitle") }}
            </n-text>
            <n-p depth="3" style="margin: 8px 0 0 0">
              {{ $t("message.importRconDesc") }}
            </n-p>
          </n-upload-dragger>
        </n-upload>
      </n-tab-pane>
      <n-tab-pane name="add" :tab="$t('button.add')">
        <n-input
          v-model:value="newRconCommand"
          size="large"
          round
          :placeholder="$t('input.rcon')"
        ></n-input>
        <n-input
          class="mt-5"
          v-model:value="newRconRemark"
          size="large"
          round
          :placeholder="$t('input.remark')"
        ></n-input>
        <n-input
          class="mt-5"
          v-model:value="newRconPlaceholder"
          size="large"
          round
          :placeholder="$t('input.placeholder')"
        ></n-input>
        <n-button
          class="mt-5"
          style="width: 100%"
          type="primary"
          @click="addRconCommand"
          strong
          secondary
        >
          {{ $t("button.add") }}
        </n-button>
      </n-tab-pane>
    </n-tabs>
  </n-modal>
  <n-drawer v-model:show="showRconDrawer" :width="502" placement="right">
    <n-drawer-content :title="t('modal.rcon')">
      <template #footer>
        <n-button type="primary" strong secondary @click="handleAddRconCommand">
          {{ $t("button.addRcon") }}
        </n-button>
      </template>
      <n-alert type="info" :show-icon="false">
        {{ $t("message.rconQuickTip") }}
      </n-alert>
      <n-divider>{{ $t("button.quickGrant") }}</n-divider>
      <div class="flex w-full items-center">
        <n-select
          :placeholder="$t('input.selectPlayer')"
          v-model:value="rconSelectedPlayer"
          filterable
          :options="rconPlayerOptions"
        />
      </div>
      <div class="flex w-full items-center mt-3 justify-between">
        <n-text>
          PlayerID: {{ rconSelectedPlayer?.split("|")[0] || "-" }}
        </n-text>
        <n-text>UserID: {{ rconSelectedPlayer?.split("|")[1] || "-" }}</n-text>
      </div>

      <div class="flex w-full items-center mt-3">
        <n-select
          :placeholder="$t('input.selectItem')"
          v-model:value="rconSelectedItem"
          filterable
          :options="rconItemOptions"
          style="flex: 1"
        />
        <n-input-number
          class="ml-2"
          v-model:value="rconCommandAmount"
          :min="1"
          :precision="0"
          style="width: 110px"
        />
        <n-button class="ml-2" type="primary" strong secondary @click="quickGiveItem">
          {{ $t("button.quickGiveItem") }}
        </n-button>
      </div>
      <div class="flex w-full items-center mt-3 justify-between">
        <n-text>ItemID: {{ rconSelectedItem || "-" }}</n-text>
        <n-text>{{ $t("input.amount") }}: {{ rconCommandAmount || 1 }}</n-text>
      </div>

      <div class="flex w-full items-center mt-3">
        <n-select
          :placeholder="$t('input.selectPal')"
          v-model:value="rconSelectedPal"
          filterable
          :options="rconPalOptions"
          style="flex: 1"
        />
        <n-input-number
          class="ml-2"
          v-model:value="rconCommandLevel"
          :min="1"
          :precision="0"
          style="width: 110px"
        />
        <n-button class="ml-2" type="primary" strong secondary @click="quickGivePal">
          {{ $t("button.quickGivePal") }}
        </n-button>
      </div>
      <div class="flex w-full items-center mt-3 justify-between">
        <n-text>PalID: {{ rconSelectedPal || "-" }}</n-text>
        <n-text>{{ $t("pal.level") }}: {{ rconCommandLevel || 1 }}</n-text>
      </div>

      <div class="flex w-full items-center mt-3">
        <n-select
          :placeholder="$t('input.selectEgg')"
          v-model:value="rconSelectedEgg"
          filterable
          :options="rconEggOptions"
          style="flex: 1"
        />
        <n-button class="ml-2" type="primary" strong secondary @click="quickGiveEgg">
          {{ $t("button.quickGiveEgg") }}
        </n-button>
      </div>
      <div class="flex w-full items-center mt-3 justify-between">
        <n-text>EggID: {{ rconSelectedEgg || "-" }}</n-text>
      </div>

      <div class="flex flex-wrap gap-2 mt-3">
        <n-button size="small" tertiary @click="quickGrantByAmount('exp')">
          {{ $t("button.quickGiveExp") }}
        </n-button>
        <n-button size="small" tertiary @click="quickGrantByAmount('relic')">
          {{ $t("button.quickGiveRelic") }}
        </n-button>
        <n-button size="small" tertiary @click="quickGrantByAmount('tech_points')">
          {{ $t("button.quickGiveTechPoints") }}
        </n-button>
        <n-button size="small" tertiary @click="quickGrantByAmount('ancient_tech_points')">
          {{ $t("button.quickGiveBossTechPoints") }}
        </n-button>
      </div>

      <pal-defender-batch-operations
        v-if="isLogin"
        :player-list="playerList"
        :guild-list="guildList"
      />

      <n-divider>{{ $t("button.rcon") }}</n-divider>
      <n-empty class="mt-3" v-if="rconCommands.length == 0"> </n-empty>
      <n-collapse class="mt-3">
        <n-collapse-item
          v-for="rconCommand in rconCommands"
          :key="rconCommand.uuid"
          :title="rconCommand.command"
          :name="rconCommand.uuid"
        >
          <template #header-extra> {{ rconCommand.remark }} </template>
          <n-input-group>
            <n-input
              round
              :placeholder="rconCommand.placeholder"
              v-model:value="rconCommandsExtra[rconCommand.uuid]"
            >
              <template #prefix>
                <n-text>{{
                  rconCommand.command + (rconCommand.placeholder ? "  +" : "")
                }}</n-text>
              </template>
            </n-input>
            <n-button
                type="info"
                ghost
                round
                @click="fillRconCommand(rconCommand)"
            >
              {{ $t("button.fill") }}
            </n-button>
            <n-button
              type="primary"
              ghost
              round
              @click="sendRconCommand(rconCommand.uuid)"
            >
              {{ $t("button.execute") }}
            </n-button>
          </n-input-group>
          <n-button
            class="mt-3"
            style="width: 100%"
            type="error"
            dashed
            @click="removeRconCommand(rconCommand.uuid)"
          >
            <template #icon>
              <n-icon>
                <DeleteFilled />
              </n-icon>
            </template>
            {{ $t("button.remove") }}
          </n-button>
        </n-collapse-item>
      </n-collapse>
    </n-drawer-content>
  </n-drawer>

  <!-- whitelist modal -->
  <n-modal
    v-model:show="showWhiteListModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 700px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.whitelist')"
    size="large"
    :bordered="false"
    :mask-closable="false"
    :close-on-esc="false"
    :segmented="segmented"
  >
    <div>
      <n-empty v-if="whiteList.length == 0"> </n-empty>
      <n-virtual-list
        v-else
        ref="virtualListInst"
        style="height: 320px"
        :item-size="42"
        :items="whiteList"
      >
        <template #default="{ item }">
          <div
            :key="item.player_uid"
            class="flex flex-col item mlr-3 mb-3"
            style="height: 42px"
          >
            <n-grid>
              <n-gi span="19">
                <n-input-group>
                  <n-input
                    v-model:value="item.name"
                    :style="{ width: '33%' }"
                    :placeholder="$t('input.nickname')"
                  />
                  <n-input
                    v-model:value="item.player_uid"
                    :style="{ width: '33%' }"
                    :placeholder="$t('input.player_uid')"
                  />
                  <n-input
                    v-model:value="item.steam_id"
                    :style="{ width: '33%' }"
                    :placeholder="$t('input.steam_id')"
                  />
                </n-input-group>
              </n-gi>
              <n-gi span="5">
                <div class="flex justify-end mr-3">
                  <n-space v-if="item.player_uid || item.steam_id">
                    <n-button
                      strong
                      secondary
                      type="primary"
                      @click="showCurrentPlayer(item.player_uid)"
                    >
                      <template #icon>
                        <n-icon><RemoveRedEyeTwotone /></n-icon>
                      </template>
                    </n-button>
                    <n-button
                      @click="removeWhiteList(item)"
                      strong
                      secondary
                      type="error"
                    >
                      <template #icon>
                        <n-icon><DeleteOutlineTwotone /></n-icon>
                      </template>
                    </n-button>
                  </n-space>
                </div>
              </n-gi>
            </n-grid>
          </div>
        </template>
      </n-virtual-list>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <n-space>
          <n-button type="primary" @click="handleAddNewWhiteList">
            {{ $t("button.addNew") }}
          </n-button>

          <n-button
            type="tertiary"
            @click="
              () => {
                showWhiteListModal = false;
              }
            "
          >
            {{ $t("button.cancel") }}
          </n-button>

          <n-button
            :disabled="whiteList.length === 0"
            @click="putWhiteList"
            strong
            secondary
            type="success"
          >
            {{ $t("button.save") }}
          </n-button>
        </n-space>
      </div>
    </template>
  </n-modal>
  <!-- backup modal -->
  <n-modal
    v-model:show="backupModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 700px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.backup')"
    size="small"
    :bordered="false"
    :mask-closable="false"
    :close-on-esc="false"
    :segmented="segmented"
  >
    <div>
      <n-empty description="empty" v-if="backupList.length == 0"> </n-empty>
      <div class="flex flex-col item mlr-3 mb-3 p-1" v-else>
        <n-date-picker
          class="mb-4"
          v-model:value="range"
          type="datetimerange"
          @confirm="getBackupListWithRange"
        />
        <n-scrollbar style="max-height: 320px">
          <n-data-table
            :columns="backupColumns"
            :data="backupList"
            :bordered="false"
          />
        </n-scrollbar>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <n-space>
          <n-button
            type="tertiary"
            @click="
              () => {
                backupModal = false;
              }
            "
          >
            {{ $t("button.close") }}
          </n-button>
        </n-space>
      </div>
    </template>
  </n-modal>
</template>
