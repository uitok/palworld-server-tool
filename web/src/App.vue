<script setup>
import { zhCN, dateZhCN, jaJP, dateJaJP, darkTheme } from "naive-ui";
import pageStore from "@/stores/model/page.js";
import { onMounted } from "vue";

const SUPPORTED_LOCALES = ["zh", "en", "ja"];

const isDarkMode = ref(
  window.matchMedia("(prefers-color-scheme: dark)").matches
);

const updateDarkMode = (e) => {
  isDarkMode.value = e.matches;
};

const themeOverrides = {
  common: {
    primaryColor: "#4098fc",
    primaryColorHover: "#4098fc",
  },
};

const locale = ref("zh");
const uiLocale = ref(zhCN);
const uiDateLocale = ref(dateZhCN);

const resolveLocale = () => {
  const savedLocale = localStorage.getItem("locale");
  return SUPPORTED_LOCALES.includes(savedLocale) ? savedLocale : "zh";
};

const applyLocale = (nextLocale) => {
  locale.value = nextLocale;
  if (nextLocale === "ja") {
    uiLocale.value = jaJP;
    uiDateLocale.value = dateJaJP;
    return;
  }
  if (nextLocale === "en") {
    uiLocale.value = null;
    uiDateLocale.value = null;
    return;
  }
  uiLocale.value = zhCN;
  uiDateLocale.value = dateZhCN;
};

const getScreenWidth = () => {
  const scrollWidth = document.documentElement.clientWidth || window.innerWidth;
  pageStore().setScreenWidth(scrollWidth);
};

const initialLocale = resolveLocale();
localStorage.setItem("locale", initialLocale);
applyLocale(initialLocale);
getScreenWidth();

onMounted(() => {
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  mediaQuery.addEventListener("change", updateDarkMode);
  isDarkMode.value = mediaQuery.matches;
  getScreenWidth();
  window.onresize = () => {
    getScreenWidth();
  };
  applyLocale(resolveLocale());
});
</script>

<template>
  <n-config-provider
    :locale="uiLocale"
    :date-locale="uiDateLocale"
    :theme-overrides="themeOverrides"
    :theme="isDarkMode ? darkTheme : null"
  >
    <n-dialog-provider>
      <n-notification-provider>
        <n-message-provider>
          <router-view />
        </n-message-provider>
      </n-notification-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>
