import dayjs from "dayjs";

export const LIVE_GRANT_ONLINE_WINDOW_MS = 80000;

export const isPlayerOnlineForLiveGrant = (lastOnline) => {
  if (!lastOnline) {
    return false;
  }
  return dayjs().diff(dayjs(lastOnline)) < LIVE_GRANT_ONLINE_WINDOW_MS;
};

export const explainPalDefenderError = (t, payload, fallback = "Unknown error") => {
  const code = payload?.error_code;
  const detail = payload?.error || payload?.message || fallback;
  const translated = {
    player_not_found: t("message.playerNotFound"),
    player_offline: t("message.playerMustBeOnline"),
    player_action_user_id_not_found: t("message.playerActionUserIdMissing"),
    paldefender_disabled: t("message.palDefenderDisabled"),
    paldefender_unconfigured: t("message.palDefenderUnconfigured"),
    paldefender_unreachable: t("message.palDefenderUnreachable"),
    paldefender_auth_failed: t("message.palDefenderAuthFailed"),
    paldefender_service_error: t("message.palDefenderServiceError"),
    paldefender_endpoint_not_found: t("message.palDefenderEndpointNotFound"),
    paldefender_preset_invalid: t("message.palDefenderPresetInvalid"),
  }[code];
  if (!translated) {
    return detail;
  }
  if (!detail || detail === translated) {
    return translated;
  }
  return `${translated}: ${detail}`;
};
