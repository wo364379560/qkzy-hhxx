package com.balabalaim.gcloud.tim.push.appdev.ui.navigation

sealed class LinLanAppRoute(val route: String) {
    data object Splash : LinLanAppRoute("splash")
    data object Login : LinLanAppRoute("login")
    data object Register : LinLanAppRoute("register")
    data object ForgotPassword : LinLanAppRoute("forgot_password")
    data object Realname : LinLanAppRoute("realname")
    data object Devices : LinLanAppRoute("devices")
    data object DeviceProtection : LinLanAppRoute("device_protection")
    data object Help : LinLanAppRoute("help")
    data object AntiFraud : LinLanAppRoute("anti_fraud")
    data object Main : LinLanAppRoute("main")
    data object Chat : LinLanAppRoute("chat/{conversationId}/{title}/{isGroup}") {
        fun create(conversationId: String, title: String, isGroup: Boolean) =
            "chat/${encode(conversationId)}/${encode(title)}/$isGroup"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }

    data object ChatSearch : LinLanAppRoute("chat_search/{conversationId}") {
        fun create(conversationId: String) =
            "chat_search/${java.net.URLEncoder.encode(conversationId, Charsets.UTF_8.name())}"
    }

    data object GroupDetail : LinLanAppRoute("group/{groupId}/{title}") {
        fun create(groupId: String, title: String) =
            "group/${encode(groupId)}/${encode(title)}"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }

    data object GroupNotice : LinLanAppRoute("group_notice/{groupId}") {
        fun create(groupId: String) =
            "group_notice/${java.net.URLEncoder.encode(groupId, Charsets.UTF_8.name())}"
    }

    data object GroupManage : LinLanAppRoute("group_manage/{groupId}") {
        fun create(groupId: String) =
            "group_manage/${java.net.URLEncoder.encode(groupId, Charsets.UTF_8.name())}"
    }

    data object GroupMembers : LinLanAppRoute("group_members/{groupId}") {
        fun create(groupId: String) =
            "group_members/${java.net.URLEncoder.encode(groupId, Charsets.UTF_8.name())}"
    }

    data object AddFriend : LinLanAppRoute("add_friend")

    data object CreatePublicGroup : LinLanAppRoute("create_public_group")

    data object FriendDetail : LinLanAppRoute("friend/{userId}/{title}") {
        fun create(userId: String, title: String) =
            "friend/${encode(userId)}/${encode(title)}"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }

    data object Report : LinLanAppRoute("report/{targetId}/{targetType}/{targetName}") {
        fun create(targetId: String, targetType: String, targetName: String) =
            "report/${encode(targetId)}/${encode(targetType)}/${encode(targetName)}"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }

    data object Forward : LinLanAppRoute("forward")
    data object MergerDetail : LinLanAppRoute("merger_detail")

    data object PickContactCard : LinLanAppRoute("pick_contact_card/{conversationId}/{isGroup}") {
        fun create(conversationId: String, isGroup: Boolean) =
            "pick_contact_card/${java.net.URLEncoder.encode(conversationId, Charsets.UTF_8.name())}/$isGroup"
    }

    data object ShareGroupCard : LinLanAppRoute("share_group_card/{groupId}/{title}") {
        fun create(groupId: String, title: String) =
            "share_group_card/${encode(groupId)}/${encode(title)}"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }

    data object EditProfile : LinLanAppRoute("edit_profile")
    data object SecurityPrivacy : LinLanAppRoute("security_privacy")
    data object ChangePassword : LinLanAppRoute("change_password")
    data object MessageNotify : LinLanAppRoute("message_notify")
    data object TeenMode : LinLanAppRoute("teen_mode")
    data object FriendRequests : LinLanAppRoute("friend_requests")
    data object Blacklist : LinLanAppRoute("blacklist")
    data object JoinedGroups : LinLanAppRoute("joined_groups")
    data object Legal : LinLanAppRoute("legal/{title}/{url}") {
        fun create(title: String, url: String) =
            "legal/${encode(title)}/${encode(url)}"

        private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name())
    }
}
