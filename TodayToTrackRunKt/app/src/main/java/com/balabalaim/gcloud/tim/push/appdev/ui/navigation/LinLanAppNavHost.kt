package com.balabalaim.gcloud.tim.push.appdev.ui.navigation

import android.widget.Toast
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanForwardJobs
import com.balabalaim.gcloud.tim.push.appdev.ui.main.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.navigation.compose.currentBackStackEntryAsState
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanNotifyRouter
import com.balabalaim.gcloud.tim.push.appdev.ui.auth.LinLanForgotPasswordScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.auth.LinLanLoginScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.auth.LinLanRegisterScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.auth.LinLanSplashScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanAddFriendScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanBlacklistScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanChatScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanChatSearchScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ChatSearchBus
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanCreateGroupScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanEditProfileScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ForwardBus
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ForwardPayload
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ForwardTargetScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanFriendDetailScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanFriendRequestsScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanGroupDetailScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanGroupManageScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanGroupMembersScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanGroupNoticeScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanJoinedGroupsScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanLegalWebScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanMainScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.MergerDetailBus
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanMergerDetailScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanMessageNotifyScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanChangePasswordScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanSecurityPrivacyScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanTeenModeScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanReportScreen
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import java.net.URLDecoder
import java.nio.charset.StandardCharsets

@Composable
fun LinLanAppNavHost() {
    val navController = rememberNavController()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    val session by LinLanAppContainer.sessionStore().sessionFlow.collectAsState(initial = null)
    LaunchedEffect(session?.userId, session?.sdkAppId) {
        val current = session
        if (current != null && current.userId.isNotBlank()) LinLanForwardJobs.restore(context, current.userId, current.sdkAppId)
    }
    val entry by navController.currentBackStackEntryAsState()
    val currentRoute = entry?.destination?.route
    LaunchedEffect(session?.isLoggedIn, session?.refreshToken.isNullOrBlank(), currentRoute) {
        val publicRoutes = setOf(LinLanAppRoute.Splash.route, LinLanAppRoute.Login.route,
            LinLanAppRoute.Register.route, LinLanAppRoute.ForgotPassword.route, LinLanAppRoute.Legal.route)
        if (session != null && session?.isLoggedIn == false && session?.refreshToken.isNullOrBlank() && currentRoute != null && currentRoute !in publicRoutes) {
            runCatching { LinLanImManager.logout() }
            navController.navigate(LinLanAppRoute.Login.route) {
                popUpTo(navController.graph.id) { inclusive = false }
                launchSingleTop = true
            }
        }
    }

    LaunchedEffect(Unit) {
        fun openPendingNotification() {
            if (LinLanNotifyRouter.consumeCall()) return
            val chat = LinLanNotifyRouter.consumeChat() ?: return
            val (peerId, isGroup, title) = chat
            val conversationId = if (isGroup) "group_$peerId" else "c2c_$peerId"
            navController.navigate(LinLanAppRoute.Chat.create(conversationId, title, isGroup)) {
                launchSingleTop = true
            }
        }
        openPendingNotification()
        LinLanNotifyRouter.events.collectLatest {
            openPendingNotification()
        }
    }

    NavHost(
        navController = navController,
        startDestination = LinLanAppRoute.Splash.route,
    ) {
        composable(LinLanAppRoute.Splash.route) {
            LinLanSplashScreen(
                onFinished = { loggedIn ->
                    navController.navigate(if (loggedIn) LinLanAppRoute.Main.route else LinLanAppRoute.Login.route) {
                        popUpTo(LinLanAppRoute.Splash.route) { inclusive = true }
                    }
                },
                onOpenLegal = { title, url ->
                    navController.navigate(LinLanAppRoute.Legal.create(title, url))
                },
            )
        }
        composable(LinLanAppRoute.Login.route) {
            LinLanLoginScreen(
                onLoggedIn = {
                    navController.navigate(LinLanAppRoute.Main.route) {
                        popUpTo(LinLanAppRoute.Login.route) { inclusive = true }
                    }
                },
                onRegister = { navController.navigate(LinLanAppRoute.Register.route) },
                onForgotPassword = { navController.navigate(LinLanAppRoute.ForgotPassword.route) },
                onOpenLegal = { title, url ->
                    navController.navigate(LinLanAppRoute.Legal.create(title, url))
                },
            )
        }
        composable(LinLanAppRoute.Register.route) {
            LinLanRegisterScreen(
                onRegistered = {
                    navController.navigate(LinLanAppRoute.Main.route) {
                        popUpTo(LinLanAppRoute.Login.route) { inclusive = true }
                    }
                },
                onBack = {
                    if (!navController.popBackStack()) {
                        navController.navigate(LinLanAppRoute.Login.route) {
                            popUpTo(LinLanAppRoute.Register.route) { inclusive = true }
                        }
                    }
                },
                onOpenLegal = { title, url ->
                    navController.navigate(LinLanAppRoute.Legal.create(title, url))
                },
            )
        }
        composable(LinLanAppRoute.ForgotPassword.route) {
            LinLanForgotPasswordScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.Realname.route) { LinLanRealnameScreen(onBack = { navController.popBackStack() }) }
        composable(LinLanAppRoute.Devices.route) { LinLanDevicesScreen(onBack = { navController.popBackStack() }) }
        composable(LinLanAppRoute.DeviceProtection.route) { LinLanDeviceProtectionScreen(onBack = { navController.popBackStack() }) }
        composable(LinLanAppRoute.Help.route) { LinLanHelpScreen(onBack = { navController.popBackStack() }) }
        composable(LinLanAppRoute.AntiFraud.route) { LinLanAntiFraudScreen(onBack = { navController.popBackStack() }) }
        composable(LinLanAppRoute.Main.route) {
            LinLanMainScreen(
                onOpenRealname = { navController.navigate(LinLanAppRoute.Realname.route) },
                onOpenDevices = { navController.navigate(LinLanAppRoute.Devices.route) },
                onOpenDeviceProtection = { navController.navigate(LinLanAppRoute.DeviceProtection.route) },
                onOpenHelp = { navController.navigate(LinLanAppRoute.Help.route) },
                onOpenAntiFraud = { navController.navigate(LinLanAppRoute.AntiFraud.route) },

                onOpenChat = { conversationId, title, isGroup ->
                    navController.navigate(LinLanAppRoute.Chat.create(conversationId, title, isGroup))
                },
                onAddFriend = { navController.navigate(LinLanAppRoute.AddFriend.route) },
                onCreatePublicGroup = { navController.navigate(LinLanAppRoute.CreatePublicGroup.route) },
                onOpenFriendDetail = { userId, title ->
                    navController.navigate(LinLanAppRoute.FriendDetail.create(userId, title))
                },
                onOpenFriendRequests = { navController.navigate(LinLanAppRoute.FriendRequests.route) },
                onOpenBlacklist = { navController.navigate(LinLanAppRoute.Blacklist.route) },
                onOpenGroups = { navController.navigate(LinLanAppRoute.JoinedGroups.route) },
                onOpenProfile = { navController.navigate(LinLanAppRoute.EditProfile.route) },
                onOpenSecurity = { navController.navigate(LinLanAppRoute.SecurityPrivacy.route) },
                onOpenNotify = { navController.navigate(LinLanAppRoute.MessageNotify.route) },
                onOpenTeen = { navController.navigate(LinLanAppRoute.TeenMode.route) },
                onOpenLegal = { title, url ->
                    navController.navigate(LinLanAppRoute.Legal.create(title, url))
                },
                onLogout = {
                    navController.navigate(LinLanAppRoute.Login.route) {
                        popUpTo(LinLanAppRoute.Main.route) { inclusive = true }
                    }
                },
            )
        }
        composable(
            route = LinLanAppRoute.Report.route,
            arguments = listOf(
                navArgument("targetId") { type = NavType.StringType },
                navArgument("targetType") { type = NavType.StringType },
                navArgument("targetName") { type = NavType.StringType },
            ),
        ) { entry ->
            LinLanReportScreen(
                targetId = URLDecoder.decode(entry.arguments?.getString("targetId").orEmpty(), StandardCharsets.UTF_8.name()),
                targetType = URLDecoder.decode(entry.arguments?.getString("targetType").orEmpty(), StandardCharsets.UTF_8.name()),
                targetName = URLDecoder.decode(entry.arguments?.getString("targetName").orEmpty(), StandardCharsets.UTF_8.name()),
                onBack = { navController.popBackStack() },
            )
        }
        composable(LinLanAppRoute.FriendRequests.route) {
            LinLanFriendRequestsScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.Blacklist.route) {
            LinLanBlacklistScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.JoinedGroups.route) {
            LinLanJoinedGroupsScreen(
                onBack = { navController.popBackStack() },
                onOpenChat = { conversationId, title, isGroup ->
                    navController.navigate(LinLanAppRoute.Chat.create(conversationId, title, isGroup))
                },
            )
        }
        composable(LinLanAppRoute.EditProfile.route) {
            LinLanEditProfileScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.SecurityPrivacy.route) {
            LinLanSecurityPrivacyScreen(
                onBack = { navController.popBackStack() },
                onOpenLegal = { title, url ->
                    navController.navigate(LinLanAppRoute.Legal.create(title, url))
                },
                onOpenChangePassword = {
                    navController.navigate(LinLanAppRoute.ChangePassword.route)
                },
                onLogout = {
                    navController.navigate(LinLanAppRoute.Login.route) {
                        popUpTo(LinLanAppRoute.Main.route) { inclusive = true }
                    }
                },
            )
        }
        composable(LinLanAppRoute.ChangePassword.route) {
            LinLanChangePasswordScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.MessageNotify.route) {
            LinLanMessageNotifyScreen(onBack = { navController.popBackStack() })
        }
        composable(LinLanAppRoute.TeenMode.route) {
            LinLanTeenModeScreen(onBack = { navController.popBackStack() })
        }
        composable(
            route = LinLanAppRoute.Legal.route,
            arguments = listOf(
                navArgument("title") { type = NavType.StringType },
                navArgument("url") { type = NavType.StringType },
            ),
        ) { entry ->
            LinLanLegalWebScreen(
                title = URLDecoder.decode(
                    entry.arguments?.getString("title").orEmpty(),
                    StandardCharsets.UTF_8.name(),
                ),
                url = URLDecoder.decode(
                    entry.arguments?.getString("url").orEmpty(),
                    StandardCharsets.UTF_8.name(),
                ),
                onBack = { navController.popBackStack() },
            )
        }
        composable(LinLanAppRoute.AddFriend.route) {
            LinLanAddFriendScreen(
                onBack = { navController.popBackStack() },
                onOpenChat = { conversationId, title, isGroup ->
                    navController.navigate(LinLanAppRoute.Chat.create(conversationId, title, isGroup))
                },
                onOpenFriendDetail = { userId, title ->
                    navController.navigate(LinLanAppRoute.FriendDetail.create(userId, title))
                },
            )
        }
        composable(LinLanAppRoute.CreatePublicGroup.route) {
            LinLanCreateGroupScreen(
                onBack = { navController.popBackStack() },
                onCreated = { groupId, title ->
                    navController.navigate(LinLanAppRoute.Chat.create("group_$groupId", title, true)) {
                        popUpTo(LinLanAppRoute.Main.route)
                    }
                },
            )
        }
        composable(
            route = LinLanAppRoute.FriendDetail.route,
            arguments = listOf(
                navArgument("userId") { type = NavType.StringType },
                navArgument("title") { type = NavType.StringType },
            ),
        ) { entry ->
            LinLanFriendDetailScreen(
                userId = entry.arguments?.getString("userId").orEmpty(),
                title = entry.arguments?.getString("title").orEmpty(),
                onBack = { navController.popBackStack() },
                onOpenChat = { conversationId, title, isGroup ->
                    navController.navigate(LinLanAppRoute.Chat.create(conversationId, title, isGroup))
                },
                onOpenReport = { targetId, targetType, targetName ->
                    navController.navigate(LinLanAppRoute.Report.create(targetId, targetType, targetName))
                },
                onDeleted = {
                    Toast.makeText(context, "已删除好友", Toast.LENGTH_SHORT).show()
                    navController.popBackStack()
                },
            )
        }
        composable(LinLanAppRoute.Forward.route) {
            ForwardTargetScreen(
                multiSelect = true,
                title = "选择会话",
                onBack = {
                    ForwardBus.pending = null
                    navController.popBackStack()
                },
                onPickedMultiple = { targets ->
                    val payload = ForwardBus.pending
                    ForwardBus.pending = null
                    if (payload == null || targets.isEmpty()) {
                        navController.popBackStack()
                        return@ForwardTargetScreen
                    }
                    val started = LinLanForwardJobs.start(context, payload, targets)
                    if (started) navController.popBackStack()
                    else Toast.makeText(context, "已有转发任务正在执行，请先查看结果", Toast.LENGTH_SHORT).show()
                },
            )
        }
        composable(LinLanAppRoute.MergerDetail.route) {
            LinLanMergerDetailScreen(
                onBack = {
                    MergerDetailBus.pending = null
                    navController.popBackStack()
                },
            )
        }
        composable(
            route = LinLanAppRoute.PickContactCard.route,
            arguments = listOf(
                navArgument("conversationId") { type = NavType.StringType },
                navArgument("isGroup") { type = NavType.BoolType },
            ),
        ) { entry ->
            val convId = URLDecoder.decode(
                entry.arguments?.getString("conversationId").orEmpty(),
                StandardCharsets.UTF_8.name(),
            )
            val isGroup = entry.arguments?.getBoolean("isGroup") ?: false
            val peer = convId.removePrefix("c2c_").removePrefix("group_")
            ForwardTargetScreen(
                onBack = { navController.popBackStack() },
                title = "选择名片",
                friendsOnly = true,
                onPicked = { receiver, _, name ->
                    val cardUserId = receiver ?: return@ForwardTargetScreen
                    scope.launch {
                        runCatching {
                            LinLanImManager.sendContactCard(
                                userId = cardUserId,
                                nickname = name,
                                receiver = if (isGroup) null else peer,
                                groupId = if (isGroup) peer else null,
                            )
                        }.onSuccess {
                            Toast.makeText(context, "名片已发送", Toast.LENGTH_SHORT).show()
                            navController.popBackStack()
                        }.onFailure {
                            Toast.makeText(context, it.message ?: "发送失败", Toast.LENGTH_SHORT).show()
                        }
                    }
                },
            )
        }
        composable(
            route = LinLanAppRoute.ShareGroupCard.route,
            arguments = listOf(
                navArgument("groupId") { type = NavType.StringType },
                navArgument("title") { type = NavType.StringType },
            ),
        ) { entry ->
            val gid = URLDecoder.decode(
                entry.arguments?.getString("groupId").orEmpty(),
                StandardCharsets.UTF_8.name(),
            )
            val gTitle = URLDecoder.decode(
                entry.arguments?.getString("title").orEmpty(),
                StandardCharsets.UTF_8.name(),
            )
            ForwardTargetScreen(
                onBack = { navController.popBackStack() },
                onPicked = { receiver, toGroupId, title ->
                    scope.launch {
                        runCatching {
                            LinLanImManager.sendGroupCard(gid, gTitle, receiver, toGroupId)
                        }.onSuccess {
                            Toast.makeText(context, "群名片已发送到 $title", Toast.LENGTH_SHORT).show()
                            navController.popBackStack()
                        }.onFailure {
                            Toast.makeText(context, it.message ?: "发送失败", Toast.LENGTH_SHORT).show()
                        }
                    }
                },
            )
        }
        composable(
            route = LinLanAppRoute.Chat.route,
            arguments = listOf(
                navArgument("conversationId") { type = NavType.StringType },
                navArgument("title") { type = NavType.StringType },
                navArgument("isGroup") { type = NavType.BoolType },
            ),
        ) { entry ->
            val conversationId = entry.arguments?.getString("conversationId").orEmpty()
            val isGroup = entry.arguments?.getBoolean("isGroup") ?: false
            val decodedConvId = URLDecoder.decode(conversationId, StandardCharsets.UTF_8.name())
            LinLanChatScreen(
                onOpenRealname = { navController.navigate(LinLanAppRoute.Realname.route) },
                conversationId = conversationId,
                title = entry.arguments?.getString("title").orEmpty(),
                isGroup = isGroup,
                onBack = { navController.popBackStack() },
                onOpenGroupDetail = { groupId, title ->
                    navController.navigate(LinLanAppRoute.GroupDetail.create(groupId, title))
                },
                onOpenGroupCard = { groupId, title ->
                    val owner = com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser
                    scope.launch {
                        try {
                            val info = LinLanImManager.fetchGroupInfo(groupId)
                            if (owner != com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) return@launch
                            if ((info?.role ?: 0) > 0) navController.navigate(LinLanAppRoute.Chat.create("group_$groupId", info?.groupName ?: title, true))
                            else navController.navigate(LinLanAppRoute.GroupDetail.create(groupId, title))
                        } catch (error: Exception) {
                            if (owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) navController.navigate(LinLanAppRoute.GroupDetail.create(groupId, title))
                        }
                    }
                },
                onOpenFriendDetail = { userId, title ->
                    navController.navigate(LinLanAppRoute.FriendDetail.create(userId, title))
                },
                onForward = { msgs, merge, mergeTitle ->
                    ForwardBus.pending = ForwardPayload(
                        messages = msgs,
                        merge = merge,
                        mergeTitle = mergeTitle,
                    )
                    navController.navigate(LinLanAppRoute.Forward.route)
                },
                onOpenMergerDetail = { msg ->
                    MergerDetailBus.pending = msg
                    navController.navigate(LinLanAppRoute.MergerDetail.route)
                },
                onPickContactCard = {
                    navController.navigate(LinLanAppRoute.PickContactCard.create(decodedConvId, isGroup))
                },
                onOpenSearch = {
                    navController.navigate(LinLanAppRoute.ChatSearch.create(decodedConvId))
                },
            )
        }
        composable(
            route = LinLanAppRoute.ChatSearch.route,
            arguments = listOf(navArgument("conversationId") { type = NavType.StringType }),
        ) { entry ->
            val encoded = entry.arguments?.getString("conversationId").orEmpty()
            val decoded = URLDecoder.decode(encoded, StandardCharsets.UTF_8.name())
            LinLanChatSearchScreen(
                conversationId = encoded,
                onBack = { navController.popBackStack() },
                onResult = { msg ->
                    ChatSearchBus.select(decoded, msg.msgID.orEmpty())
                    navController.popBackStack()
                },
            )
        }
        composable(
            route = LinLanAppRoute.GroupDetail.route,
            arguments = listOf(
                navArgument("groupId") { type = NavType.StringType },
                navArgument("title") { type = NavType.StringType },
            ),
        ) { entry ->
            val groupId = entry.arguments?.getString("groupId").orEmpty()
            val title = entry.arguments?.getString("title").orEmpty()
            val decodedGid = URLDecoder.decode(groupId, StandardCharsets.UTF_8.name())
            val decodedTitle = URLDecoder.decode(title, StandardCharsets.UTF_8.name())
            LinLanGroupDetailScreen(
                groupId = groupId,
                title = title,
                onBack = { navController.popBackStack() },
                onOpenChat = { conversationId, t, isGroup ->
                    navController.navigate(LinLanAppRoute.Chat.create(conversationId, t, isGroup))
                },
                onOpenNotice = { gid ->
                    navController.navigate(LinLanAppRoute.GroupNotice.create(gid))
                },
                onOpenManage = { gid ->
                    navController.navigate(LinLanAppRoute.GroupManage.create(gid))
                },
                onOpenMembers = { gid ->
                    navController.navigate(LinLanAppRoute.GroupMembers.create(gid))
                },
                onOpenReport = { targetId, targetType, targetName ->
                    navController.navigate(LinLanAppRoute.Report.create(targetId, targetType, targetName))
                },
                onShareGroupCard = {
                    navController.navigate(LinLanAppRoute.ShareGroupCard.create(decodedGid, decodedTitle))
                },
                onLeftGroup = {
                    if (!navController.popBackStack(LinLanAppRoute.Main.route, inclusive = false)) {
                        navController.navigate(LinLanAppRoute.Main.route) {
                            popUpTo(LinLanAppRoute.Splash.route) { inclusive = true }
                        }
                    }
                },
            )
        }
        composable(
            route = LinLanAppRoute.GroupNotice.route,
            arguments = listOf(navArgument("groupId") { type = NavType.StringType }),
        ) { entry ->
            LinLanGroupNoticeScreen(
                groupId = entry.arguments?.getString("groupId").orEmpty(),
                onBack = { navController.popBackStack() },
            )
        }
        composable(
            route = LinLanAppRoute.GroupManage.route,
            arguments = listOf(navArgument("groupId") { type = NavType.StringType }),
        ) { entry ->
            LinLanGroupManageScreen(
                groupId = entry.arguments?.getString("groupId").orEmpty(),
                onBack = { navController.popBackStack() },
                onOpenMemberDetail = { userId, name ->
                    navController.navigate(LinLanAppRoute.FriendDetail.create(userId, name))
                },
            )
        }
        composable(
            route = LinLanAppRoute.GroupMembers.route,
            arguments = listOf(navArgument("groupId") { type = NavType.StringType }),
        ) { entry ->
            LinLanGroupMembersScreen(
                groupId = entry.arguments?.getString("groupId").orEmpty(),
                onBack = { navController.popBackStack() },
                onOpenMemberDetail = { userId, name ->
                    navController.navigate(LinLanAppRoute.FriendDetail.create(userId, name))
                },
            )
        }
    }
    ForwardResultsHost(session?.userId.orEmpty())
    com.balabalaim.gcloud.tim.push.appdev.ui.main.VideoResultsHost(session?.userId.orEmpty())

}
