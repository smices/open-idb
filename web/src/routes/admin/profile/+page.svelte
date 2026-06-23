<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
    import { api } from "$lib/api";
    import { t } from "$lib/i18n";
    import { authUser } from "$lib/stores";
    import Toast from "$lib/components/ui/Toast.svelte";

    let displayName = "";
    let currentPassword = "";
    let newPassword = "";
    let confirmPassword = "";
    let profileSubmitting = false;
    let passwordSubmitting = false;
    let profileError = "";
    let profileSuccess = "";
    let passwordError = "";
    let passwordSuccess = "";
    let profileLoadedFor = "";

    const maskedIdentifier = (value?: string): string => {
        if (!value) return "-";
        if (value.length <= 8) return value;
        return `${value.slice(0, 4)}...${value.slice(-4)}`;
    };

    const saveProfile = async () => {
        profileError = "";
        profileSuccess = "";
        if (!displayName.trim()) {
            profileError = t("profile.displayNameRequired");
            return;
        }
        profileSubmitting = true;
        try {
            const next = await api.updateAdminMe({
                display_name: displayName.trim(),
            });
            authUser.update((current) =>
                current
                    ? { ...current, display_name: next.display_name }
                    : current,
            );
            profileSuccess = t("profile.profileUpdateSuccess");
        } catch {
            profileError = t("profile.profileUpdateFailed");
        } finally {
            profileSubmitting = false;
        }
    };

    const savePassword = async () => {
        passwordError = "";
        passwordSuccess = "";
        if (!currentPassword || !newPassword || !confirmPassword) {
            passwordError = t("profile.currentPasswordRequired");
            return;
        }
        if (newPassword.length < 12) {
            passwordError = t("profile.passwordTooShort");
            return;
        }
        if (newPassword !== confirmPassword) {
            passwordError = t("profile.passwordMismatch");
            return;
        }
        passwordSubmitting = true;
        try {
            await api.updateAdminPassword({
                current_password: currentPassword,
                new_password: newPassword,
            });
            passwordSuccess = t("profile.updateSuccess");
            currentPassword = "";
            newPassword = "";
            confirmPassword = "";
        } catch {
            passwordError = t("profile.currentPasswordError");
        } finally {
            passwordSubmitting = false;
        }
    };

    $: user = $authUser;
    $: if (user && user.id !== profileLoadedFor) {
        profileLoadedFor = user.id;
        displayName = user.display_name || "";
    }
</script>

<svelte:head>
    <title>{t("profile.title")}</title>
</svelte:head>

<section class="grid gap-4 xl:grid-cols-2">
    <Toast message={profileSuccess || passwordSuccess} />
    <form
        class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3"
        on:submit|preventDefault={saveProfile}
    >
        <h2 class="text-lg font-semibold">{t("profile.editProfile")}</h2>
        {#if profileError}
            <aside class="alert preset-tonal-error" role="alert">
                <p>{profileError}</p>
            </aside>
        {/if}

        <label class="block">
            <span class="text-sm text-surface-500"
                >{t("users.displayName")}</span
            >
            <input
                class="input w-full"
                type="text"
                bind:value={displayName}
                required
            />
        </label>

        <dl class="grid gap-3 text-sm">
            <div
                class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2"
            >
                <dt class="text-surface-500">{t("profile.userId")}</dt>
                <dd class="font-medium">{maskedIdentifier(user?.id)}</dd>
            </div>
            <div
                class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2"
            >
                <dt class="text-surface-500">{t("users.username")}</dt>
                <dd class="font-medium">{user?.username || "-"}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
                <dt class="text-surface-500">{t("profile.entityId")}</dt>
                <dd class="font-medium">{maskedIdentifier(user?.entity_id)}</dd>
            </div>
        </dl>

        <button
            class="btn btn-xs preset-filled-primary-500"
            type="submit"
            disabled={profileSubmitting}
        >
            {profileSubmitting ? t("common.loading") : t("profile.saveProfile")}
        </button>
    </form>

    <form
        id="password"
        class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3"
        on:submit|preventDefault={savePassword}
    >
        <div>
            <h2 class="text-lg font-semibold">{t("profile.changePassword")}</h2>
            <p class="mt-1 text-sm text-surface-500">
                {t("profile.passwordPolicyHint")}
            </p>
        </div>

        {#if passwordError}
            <aside class="alert preset-tonal-error" role="alert">
                <p>{passwordError}</p>
            </aside>
        {/if}

        <label class="block">
            <span class="text-sm text-surface-500"
                >{t("profile.currentPassword")}</span
            >
            <input
                class="input w-full"
                type="password"
                bind:value={currentPassword}
                autocomplete="current-password"
                required
            />
        </label>
        <label class="block">
            <span class="text-sm text-surface-500"
                >{t("profile.newPassword")}</span
            >
            <input
                class="input w-full"
                type="password"
                bind:value={newPassword}
                autocomplete="new-password"
                required
            />
        </label>
        <label class="block">
            <span class="text-sm text-surface-500"
                >{t("profile.confirmPassword")}</span
            >
            <input
                class="input w-full"
                type="password"
                bind:value={confirmPassword}
                autocomplete="new-password"
                required
            />
        </label>

        <button
            class="btn btn-xs preset-filled-primary-500"
            type="submit"
            disabled={passwordSubmitting}
        >
            {passwordSubmitting
                ? t("common.loading")
                : t("profile.updatePassword")}
        </button>
    </form>
</section>
