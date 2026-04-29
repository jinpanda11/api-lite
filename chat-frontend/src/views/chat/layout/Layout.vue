<script setup lang='ts'>
import { computed, onMounted } from 'vue'
import { NLayout, NLayoutContent, useDialog, useMessage } from 'naive-ui'
import { useRouter ,useRoute } from 'vue-router'
import Sider from './sider/index.vue'
import Permission from './Permission.vue'
import { useBasicLayout } from '@/hooks/useBasicLayout'
import { gptConfigStore, gptServerStore, homeStore, useAppStore, useAuthStore, useChatStore } from '@/store'
import { aiSider,aiFooter} from '@/views/mj' 
import aiMobileMenu from '@/views/mj/aiMobileMenu.vue'; 
import { t } from '@/locales'
import { mlog, openaiSetting } from '@/api'
import { get } from '@/utils/request'
import { isObject } from '@/utils/is'

const router = useRouter()
const appStore = useAppStore()
const chatStore = useChatStore()
const authStore = useAuthStore()

const rt = useRoute();
const ms = useMessage();
openaiSetting( rt.query, ms )
// Auto-configure API base URL from current site origin
if (!gptServerStore.myData.OPENAI_API_BASE_URL) {
    gptServerStore.setMyData({ OPENAI_API_BASE_URL: window.location.origin })
}
if(rt.name =='GPTs'){
  let model= `gpt-4-gizmo-${rt.params.gid.toString()}`  ;
  gptConfigStore.setMyData({model:model});
  ms.success(`GPTs ${t('mj.modleSuccess')}`);
}else if(rt.name=='Setting'){ 
  openaiSetting( rt.query,ms );
  if(isObject( rt.query ))  ms.success( t('mj.setingSuccess') ); 
}else if(rt.name=='Model'){ 
  let model= `${rt.params.gid.toString()}`  ;
  gptConfigStore.setMyData({model:model});
  ms.success( t('mj.modleSuccess') );
}

 

router.replace({ name: 'Chat', params: { uuid: chatStore.active } })
homeStore.setMyData({local:'Chat'});
const { isMobile } = useBasicLayout()

// Auto-authorize API key
const dialog = useDialog()
onMounted(() => {
	setTimeout(() => {
		if (!gptServerStore.myData.OPENAI_API_KEY) {
			dialog.create({
				title: 'API Key 授权',
				content: '是否自动使用本站 API Key 填入设置？选择是，系统将自动为您配置密钥。',
				positiveText: '是，自动填入',
				negativeText: '否',
				closable: false,
				closeOnEsc: false,
				maskClosable: false,
				onPositiveClick: async () => {
					try {
						const res = await get<{ key: string }>({ url: '/token/chat-key' })
						if (res.data?.key) {
							gptServerStore.setMyData({ OPENAI_API_KEY: res.data.key })
							ms.success('API Key 已自动填入！')
							return
						}
					} catch (e) {
						// ignore
					}
					ms.error('获取 API Key 失败，请前往设置页面手动配置')
				},
			})
		}
	}, 800)
})

const collapsed = computed(() => appStore.siderCollapsed)

const needPermission = computed(() => {
//mlog( 'Layout token',  authStore.token   )
   
 return  !!authStore.session?.auth && !authStore.token
})

const getMobileClass = computed(() => {
  if (isMobile.value)
    return ['rounded-none', 'shadow-none']
  return [ 'shadow-md', 'dark:border-neutral-800'] //'border', 'rounded-md',
})

const getContainerClass = computed(() => {
  return [
    'h-full',
    { 'abc': !isMobile.value && !collapsed.value },
  ]
}) 
</script>

<template>
  <div class="  dark:bg-[#24272e] transition-all p-0"  :class="[isMobile ? 'h55' : 'h-full' ]">
    <div class="h-full overflow-hidden" :class="getMobileClass">
      <NLayout class="z-40 transition" :class="getContainerClass" has-sider>
        <aiSider v-if="!isMobile"/>
        <Sider />
        <NLayoutContent class="h-full">
          <RouterView v-slot="{ Component, route }">
            <component :is="Component" :key="route.fullPath" />
          </RouterView>
        </NLayoutContent>
      </NLayout>
    </div>
    <Permission :visible="needPermission" />
  </div>
   <aiMobileMenu v-if="isMobile"   /> 

  <aiFooter/>
</template>

<style  >
.h55{
  height: calc(100% - 55px);
}
</style>
