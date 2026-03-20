import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("@/views/Home.vue"),
    },
    {
      path: "/ops",
      name: "operations",
      component: () => import("@/views/OperationsOverview.vue"),
    },
    {
      path: "/players",
      name: "players",
      component: () => import("@/views/PlayersOverview.vue"),
    },
    {
      path: "/paldefender",
      name: "paldefender",
      component: () => import("@/views/PalDefenderOverview.vue"),
    },
  ],
});

export default router;
