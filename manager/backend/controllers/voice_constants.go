package controllers

import "strings"

// VoiceOption represents a voice option.
type VoiceOption struct {
	Value string `json:"value"` // voice value
	Label string `json:"label"` // voice display name
}

// VoiceOptions defines voice options for each provider.
// Per Volcengine Doubao TTS documentation: https://www.volcengine.com/docs/6561/97465?lang=zh
// and Doubao WebSocket documentation: https://www.volcengine.com/docs/6561/1257544?lang=zh
var VoiceOptions = map[string][]VoiceOption{
	// Edge TTS voice list (Chinese)
	// Reference: https://blog.csdn.net/u012917925/article/details/134683773
	"edge": {
		{Value: "zh-CN-XiaoxiaoNeural", Label: "Xiaoxiao (female)"},
		{Value: "zh-CN-YunxiNeural", Label: "Yunxi (male)"},
		{Value: "zh-CN-YunyangNeural", Label: "Yunyang (male)"},
		{Value: "zh-CN-XiaoyiNeural", Label: "Xiaoyi (female)"},
		{Value: "zh-CN-YunjianNeural", Label: "Yunjian (male)"},
		{Value: "zh-CN-YunxiaNeural", Label: "Yunxia (male)"},
		{Value: "zh-CN-YunhaoNeural", Label: "Yunhao (male)"},
		{Value: "zh-CN-XiaohanNeural", Label: "Xiaohan (female)"},
		{Value: "zh-CN-XiaomoNeural", Label: "Xiaomo (female)"},
		{Value: "zh-CN-XiaoxuanNeural", Label: "Xiaoxuan (female)"},
		{Value: "zh-CN-XiaoruiNeural", Label: "Xiaorui (female)"},
		{Value: "zh-CN-XiaoshuangNeural", Label: "Xiaoshuang (female)"},
		{Value: "zh-CN-XiaoyanNeural", Label: "Xiaoyan (female)"},
		{Value: "zh-CN-XiaoyouNeural", Label: "Xiaoyou (female)"},
		{Value: "zh-CN-XiaozhenNeural", Label: "Xiaozhen (female)"},
		{Value: "zh-CN-YunfengNeural", Label: "Yunfeng (male)"},
		{Value: "zh-CN-YunyeNeural", Label: "Yunye (male)"},
		{Value: "zh-CN-YunzeNeural", Label: "Yunze (male)"},
	},

	// Microsoft TTS voice list (Chinese)
	"microsoft": {
		{Value: "zh-CN-XiaoxiaoNeural", Label: "Xiaoxiao (female)"},
		{Value: "zh-CN-YunxiNeural", Label: "Yunxi (male)"},
		{Value: "zh-CN-YunyangNeural", Label: "Yunyang (male)"},
		{Value: "zh-CN-XiaoyiNeural", Label: "Xiaoyi (female)"},
		{Value: "zh-CN-YunjianNeural", Label: "Yunjian (male)"},
		{Value: "zh-CN-YunxiaNeural", Label: "Yunxia (male)"},
		{Value: "zh-CN-YunhaoNeural", Label: "Yunhao (male)"},
		{Value: "zh-CN-XiaohanNeural", Label: "Xiaohan (female)"},
		{Value: "zh-CN-XiaomoNeural", Label: "Xiaomo (female)"},
		{Value: "zh-CN-XiaoxuanNeural", Label: "Xiaoxuan (female)"},
		{Value: "zh-CN-XiaoruiNeural", Label: "Xiaorui (female)"},
		{Value: "zh-CN-XiaoshuangNeural", Label: "Xiaoshuang (female)"},
		{Value: "zh-CN-XiaoyanNeural", Label: "Xiaoyan (female)"},
		{Value: "zh-CN-XiaoyouNeural", Label: "Xiaoyou (female)"},
		{Value: "zh-CN-XiaozhenNeural", Label: "Xiaozhen (female)"},
		{Value: "zh-CN-YunfengNeural", Label: "Yunfeng (male)"},
		{Value: "zh-CN-YunyeNeural", Label: "Yunye (male)"},
		{Value: "zh-CN-YunzeNeural", Label: "Yunze (male)"},
	},

	// Doubao TTS voice list (HTTP interface)
	// Reference: https://www.volcengine.com/docs/6561/97465?lang=zh
	"doubao": {
		{Value: "BV700_V2_streaming", Label: "Cancan 2.0"},
		{Value: "BV705_streaming", Label: "Yangyang"},
		{Value: "BV701_V2_streaming", Label: "Qingcang 2.0"},
		{Value: "BV001_V2_streaming", Label: "General Female 2.0"},
		{Value: "BV700_streaming", Label: "Cancan"},
		{Value: "BV406_V2_streaming", Label: "Ultra Natural-Zizi 2.0"},
		{Value: "BV406_streaming", Label: "Ultra Natural-Zizi"},
		{Value: "BV407_V2_streaming", Label: "Ultra Natural-Ranran 2.0"},
		{Value: "BV407_streaming", Label: "Ultra Natural-Ranran"},
		{Value: "BV001_streaming", Label: "General Female"},
		{Value: "BV002_streaming", Label: "General Male"},
		{Value: "BV701_streaming", Label: "Qingcang"},
		{Value: "BV119_streaming", Label: "General Zhui Xu"},
		{Value: "BV102_streaming", Label: "Elegant Youth"},
		{Value: "BV113_streaming", Label: "Sweet Princess"},
		{Value: "BV115_streaming", Label: "Ancient Style Princess"},
		{Value: "BV007_streaming", Label: "Warm Female"},
		{Value: "BV056_streaming", Label: "Sunny Male"},
		{Value: "BV005_streaming", Label: "Lively Female"},
		{Value: "BV051_streaming", Label: "Cute Kid"},
		{Value: "BV034_streaming", Label: "Intellectual Sister-Bilingual"},
		{Value: "BV033_streaming", Label: "Gentle Guy"},
		{Value: "BV021_streaming", Label: "Northeast Buddy"},
		{Value: "BV019_streaming", Label: "Chongqing Boy"},
		{Value: "BV213_streaming", Label: "Guangxi Brother"},
		{Value: "BV503_streaming", Label: "Energetic Female-Ariana"},
		{Value: "BV504_streaming", Label: "Energetic Male-Jackson"},
		{Value: "BV522_streaming", Label: "Elegant Girl"},
		{Value: "BV524_streaming", Label: "Japanese Male"},
		{Value: "BV104_streaming", Label: "Gentle Lady"},
		{Value: "BV004_streaming", Label: "Cheerful Youth"},
		{Value: "BV009_streaming", Label: "Intellectual Female"},
		{Value: "BV008_streaming", Label: "Warm Male"},
		{Value: "BV064_streaming", Label: "Little Lolita"},
		{Value: "BV437_streaming", Label: "Commentator-Multi Emotion"},
		{Value: "BV511_streaming", Label: "Languid Female-Ava"},
		{Value: "BV040_streaming", Label: "Warm Female-Anna"},
		{Value: "BV138_streaming", Label: "Emotional Female-Lawrence"},
		{Value: "BV704_streaming", Label: "Dialect Cancan"},
		{Value: "BV702_streaming", Label: "Stefan"},
		{Value: "BV421_streaming", Label: "Genius Girl"},
	},

	// Doubao WebSocket TTS voice list
	// Per official documentation "Voice List":
	// https://www.volcengine.com/docs/6561/1257544?lang=zh
	// This maintains commonly used official online voice candidates for this project.
	// Note: The voice list is for display purposes only; model/resource_id binding is not enforced by voice name.
	// Actual availability depends on resources activated in the Volcengine console for the current appid/access_token.

	"doubao_ws": {
		// Female voices
		{Value: "zh_female_cancan_mars_bigtts", Label: "Cancan / Shiny (female)"},
		{Value: "zh_female_vv_uranus_bigtts", Label: "Vivi 2.0 (female)"},
		{Value: "zh_female_vv_jupiter_bigtts", Label: "Vivi O (female)"},
		{Value: "zh_female_xiaohe_jupiter_bigtts", Label: "Xiaohe O (female)"},
		{Value: "saturn_zh_female_cancan_tob", Label: "Intellectual Cancan (female)"},
		{Value: "saturn_zh_female_keainvsheng_tob", Label: "Cute Girl (female)"},
		{Value: "saturn_zh_female_tiaopigongzhu_tob", Label: "Playful Princess (female)"},
		{Value: "zh_female_xiaohe_uranus_bigtts", Label: "Xiaohe (female)"},
		{Value: "zh_female_tianmeitaozi_mars_bigtts", Label: "Sweet Peach (female)"},
		{Value: "zh_female_wanwanxiaohe_moon_bigtts", Label: "Wanwan Xiaohe (female)"},
		{Value: "zh_female_qinqienvsheng_moon_bigtts", Label: "Warm Female (female)"},
		{Value: "zh_female_vv_mars_bigtts", Label: "Vivi (female)"},
		{Value: "zh_female_tianmeixiaoyuan_moon_bigtts", Label: "Sweet Xiaoyuan (female)"},
		{Value: "zh_female_qingchezizi_moon_bigtts", Label: "Clear Zizi (female)"},
		{Value: "zh_female_kailangjiejie_moon_bigtts", Label: "Cheerful Sister (female)"},
		{Value: "zh_female_tianmeiyueyue_moon_bigtts", Label: "Sweet Yueyue (female)"},
		{Value: "zh_female_xinlingjitang_moon_bigtts", Label: "Heartwarming (female)"},
		{Value: "zh_female_zhixingnvsheng_mars_bigtts", Label: "Intellectual Female (female)"},
		{Value: "zh_female_wenroushunv_mars_bigtts", Label: "Gentle Lady (female)"},
		{Value: "zh_female_wenrouxiaoya_moon_bigtts", Label: "Gentle Xiaoya (female)"},
		{Value: "zh_female_linjianvhai_moon_bigtts", Label: "Girl Next Door (female)"},
		{Value: "zh_female_shuangkuaisisi_moon_bigtts", Label: "Crisp Sisi / Skye (female)"},
		{Value: "zh_female_gaolengyujie_moon_bigtts", Label: "Cool Elder Sister (female)"},
		{Value: "zh_female_meilinvyou_moon_bigtts", Label: "Charming Girlfriend (female)"},
		{Value: "zh_female_sajiaonvyou_moon_bigtts", Label: "Gentle Girlfriend-Coquettish (female)"},
		{Value: "zh_female_yuanqinvyou_moon_bigtts", Label: "Coquettish Junior (female)"},
		{Value: "ICL_zh_female_wenrounvshen_239eff5e8ffa_tob", Label: "Gentle Goddess (female)"},
		{Value: "ICL_zh_female_chunzhenshaonv_e588402fb8ad_tob", Label: "Innocent Girl (female)"},
		{Value: "ICL_zh_female_jinglingxiaodao_1beb294a9e3e_tob", Label: "Sprite Guide (female)"},
		{Value: "ICL_zh_female_yilin_tob", Label: "Caring Little Sister (female)"},
		{Value: "ICL_zh_female_chengshujiejie_tob", Label: "Mature Sister (female)"},
		{Value: "ICL_zh_female_bingjiaojiejie_tob", Label: "Yandere Sister (female)"},
		{Value: "ICL_zh_female_wumeiyujie_tob", Label: "Alluring Elder Sister (female)"},
		{Value: "ICL_zh_female_aojiaonvyou_tob", Label: "Tsundere Girlfriend (female)"},
		{Value: "ICL_zh_female_tiexinnvyou_tob", Label: "Caring Girlfriend (female)"},
		{Value: "ICL_zh_female_xingganyujie_tob", Label: "Sensual Elder Sister (female)"},
		{Value: "ICL_zh_female_lixingyuanzi_cs_tob", Label: "Rational Yuanzi (customer service female)"},
		{Value: "ICL_zh_female_wuxi_tob", Label: "Energetic Sweet Girl (female)"},
		{Value: "ICL_zh_female_zhixingwenwan_tob", Label: "Intellectual Graceful (female)"},

		// Male voices
		{Value: "saturn_zh_male_shuanglangshaonian_tob", Label: "Cheerful Youth (male)"},
		{Value: "saturn_zh_male_tiancaitongzhuo_tob", Label: "Genius Classmate (male)"},
		{Value: "zh_male_yunzhou_jupiter_bigtts", Label: "Yunzhou O (male)"},
		{Value: "zh_male_xiaotian_jupiter_bigtts", Label: "Xiaotian O (male)"},
		{Value: "zh_male_m191_uranus_bigtts", Label: "Yunzhou (male)"},
		{Value: "zh_male_taocheng_uranus_bigtts", Label: "Xiaotian (male)"},
		{Value: "en_male_tim_uranus_bigtts", Label: "Tim (English male)"},
		{Value: "zh_male_yangguangqingnian_moon_bigtts", Label: "Sunny Youth (male)"},
		{Value: "zh_male_qingshuangnanda_mars_bigtts", Label: "Refreshing College Guy (male)"},
		{Value: "zh_male_wenrouxiaoge_mars_bigtts", Label: "Gentle Guy (male)"},
		{Value: "zh_male_qingcang_mars_bigtts", Label: "Qingcang (male)"},
		{Value: "zh_male_ruyaqingnian_mars_bigtts", Label: "Elegant Youth (male)"},
		{Value: "zh_male_jieshuoxiaoming_moon_bigtts", Label: "Commentator Xiaoming (male)"},
		{Value: "zh_male_linjiananhai_moon_bigtts", Label: "Boy Next Door (male)"},
		{Value: "zh_male_yuanboxiaoshu_moon_bigtts", Label: "Knowledgeable Uncle (male)"},
		{Value: "zh_male_wennuanahu_moon_bigtts", Label: "Warm Ahu / Alvin (male)"},
		{Value: "zh_male_shaonianzixin_moon_bigtts", Label: "Youth Zixin / Brayan (male)"},
		{Value: "zh_male_beijingxiaoye_moon_bigtts", Label: "Beijing Gentleman (male)"},
		{Value: "zh_male_jingqiangkanye_moon_bigtts", Label: "Beijing Accent / Harmony (male)"},
		{Value: "zh_male_guozhoudege_moon_bigtts", Label: "Guangzhou Dege (male)"},
		{Value: "zh_male_haoyuxiaoge_moon_bigtts", Label: "Haoyu Guy (male)"},
		{Value: "zh_male_shenyeboke_moon_bigtts", Label: "Late Night Podcast (male)"},
		{Value: "zh_male_aojiaobazong_moon_bigtts", Label: "Arrogant CEO (male)"},
		{Value: "zh_male_dongfanghaoran_moon_bigtts", Label: "Dongfang Haoran (male)"},
		{Value: "zh_male_M100_conversation_wvae_bigtts", Label: "Graceful Gentleman / Lucas (male)"},
		{Value: "zh_male_xudong_conversation_wvae_bigtts", Label: "Happy Xiaodong / Daniel (male)"},
		{Value: "zh_male_qingyiyuxuan_mars_bigtts", Label: "Sunny Achen (male)"},
		{Value: "en_male_jason_conversation_wvae_bigtts", Label: "Cheerful Senior (male)"},
		{Value: "ICL_zh_male_lengkugege_v1_tob", Label: "Cool Brother (male)"},
		{Value: "ICL_zh_male_shenmi_v1_tob", Label: "Clever Guy (male)"},
		{Value: "ICL_zh_male_BV705_streaming_cs_tob", Label: "Yangyang (male)"},
		{Value: "ICL_zh_male_menyoupingxiaoge_ffed9fc2fee7_tob", Label: "Mysterious Quiet Guy (male)"},
		{Value: "ICL_zh_male_anrenqinzhu_cd62e63dcdab_tob", Label: "Dark Blade Lord (male)"},
		{Value: "ICL_zh_male_guaogongzi_v1_tob", Label: "Lone Noble (male)"},
		{Value: "ICL_zh_male_bingruogongzi_tob", Label: "Frail Noble (male)"},
		{Value: "ICL_zh_male_bingjiaodidi_tob", Label: "Yandere Little Brother (male)"},
		{Value: "ICL_zh_male_aomanshaoye_tob", Label: "Arrogant Young Master (male)"},
		{Value: "ICL_zh_male_chunzhenxuedi_tob", Label: "Innocent Junior (male)"},
		{Value: "ICL_zh_male_yourougongzi_tob", Label: "Gentle Noble (male)"},
		{Value: "ICL_zh_male_tiexinnanyou_tob", Label: "Caring Boyfriend (male)"},
		{Value: "ICL_zh_male_shaonianjiangjun_tob", Label: "Young General (male)"},
		{Value: "ICL_zh_male_bingjiaogege_tob", Label: "Yandere Brother (male)"},
		{Value: "ICL_zh_male_xuebanantongzhuo_tob", Label: "Top Student Male Classmate (male)"},
		{Value: "ICL_zh_male_youmoshushu_tob", Label: "Humorous Uncle (male)"},
		{Value: "ICL_zh_male_wenrounantongzhuo_tob", Label: "Gentle Male Classmate (male)"},
		{Value: "ICL_zh_male_youmodaye_tob", Label: "Humorous Elder (male)"},
		{Value: "ICL_zh_male_shenmifashi_tob", Label: "Mysterious Mage (male)"},
		{Value: "ICL_zh_male_lengjunshangsi_tob", Label: "Cold-faced Boss (male)"},
		{Value: "ICL_en_male_michael_tob", Label: "Michael (American English male)"},

		// IP/Character voices
		{Value: "zh_male_lubanqihao_mars_bigtts", Label: "Luban No.7 (male)"},
		{Value: "zh_female_yangmi_mars_bigtts", Label: "Lin Xiao (female)"},
		{Value: "zh_female_linzhiling_mars_bigtts", Label: "Lingling Sister (female)"},
		{Value: "zh_female_jiyejizi2_mars_bigtts", Label: "Kasukabe Sister (female)"},
		{Value: "zh_male_tangseng_mars_bigtts", Label: "Tang Monk (male)"},
		{Value: "zh_male_zhubajie_mars_bigtts", Label: "Zhu Bajie (male)"},
		{Value: "zh_female_naying_mars_bigtts", Label: "Candid Yingzi (female)"},
		{Value: "zh_female_leidian_mars_bigtts", Label: "Female Thunder God (female)"},
		{Value: "zh_male_sunwukong_mars_bigtts", Label: "Monkey King (male)"},
		{Value: "zh_male_xionger_mars_bigtts", Label: "Xiong Er (male)"},
		{Value: "zh_female_peiqi_mars_bigtts", Label: "Peppa Pig (female)"},
		{Value: "zh_female_yingtaowanzi_mars_bigtts", Label: "Cherry Meatball (female)"},
		{Value: "zh_male_silang_mars_bigtts", Label: "Silang (male)"},
	},

	// Minimax TTS voice list
	// Reference: https://www.minimaxi.com/document/guides/tts-model
	"minimax": {
		// Chinese (Mandarin)
		{Value: "male-qn-qingse", Label: "Youthful Male Voice"},
		{Value: "male-qn-jingying", Label: "Elite Male Voice"},
		{Value: "male-qn-badao", Label: "Domineering Male Voice"},
		{Value: "male-qn-daxuesheng", Label: "College Student Male Voice"},
		{Value: "female-shaonv", Label: "Young Girl Voice"},
		{Value: "female-yujie", Label: "Cool Elder Sister Voice"},
		{Value: "female-chengshu", Label: "Mature Female Voice"},
		{Value: "female-tianmei", Label: "Sweet Female Voice"},
		{Value: "male-qn-qingse-jingpin", Label: "Youthful Male Voice-beta"},
		{Value: "male-qn-jingying-jingpin", Label: "Elite Male Voice-beta"},
		{Value: "male-qn-badao-jingpin", Label: "Domineering Male Voice-beta"},
		{Value: "male-qn-daxuesheng-jingpin", Label: "College Student Male Voice-beta"},
		{Value: "female-shaonv-jingpin", Label: "Young Girl Voice-beta"},
		{Value: "female-yujie-jingpin", Label: "Cool Elder Sister Voice-beta"},
		{Value: "female-chengshu-jingpin", Label: "Mature Female Voice-beta"},
		{Value: "female-tianmei-jingpin", Label: "Sweet Female Voice-beta"},
		{Value: "clever_boy", Label: "Smart Boy"},
		{Value: "cute_boy", Label: "Cute Boy"},
		{Value: "lovely_girl", Label: "Lovely Girl"},
		{Value: "cartoon_pig", Label: "Cartoon Pig Xiaoqi"},
		{Value: "bingjiao_didi", Label: "Yandere Little Brother"},
		{Value: "junlang_nanyou", Label: "Handsome Boyfriend"},
		{Value: "chunzhen_xuedi", Label: "Innocent Junior"},
		{Value: "lengdan_xiongzhang", Label: "Cool Senior"},
		{Value: "badao_shaoye", Label: "Domineering Young Master"},
		{Value: "tianxin_xiaoling", Label: "Sweet Xiaoling"},
		{Value: "qiaopi_mengmei", Label: "Playful Cute Girl"},
		{Value: "wumei_yujie", Label: "Alluring Elder Sister"},
		{Value: "diadia_xuemei", Label: "Coquettish Junior Girl"},
		{Value: "danya_xuejie", Label: "Elegant Senior Girl"},
		{Value: "Chinese (Mandarin)_Reliable_Executive", Label: "Composed Executive"},
		{Value: "Chinese (Mandarin)_News_Anchor", Label: "News Female Voice"},
		{Value: "Chinese (Mandarin)_Mature_Woman", Label: "Tsundere Elder Sister"},
		{Value: "Chinese (Mandarin)_Unrestrained_Young_Man", Label: "Unrestrained Youth"},
		{Value: "Arrogant_Miss", Label: "Arrogant Miss"},
		{Value: "Robot_Armor", Label: "Robot Armor"},
		{Value: "Chinese (Mandarin)_Kind-hearted_Antie", Label: "Kind-hearted Auntie"},
		{Value: "Chinese (Mandarin)_HK_Flight_Attendant", Label: "HK Flight Attendant"},
		{Value: "Chinese (Mandarin)_Humorous_Elder", Label: "Humorous Elder"},
		{Value: "Chinese (Mandarin)_Gentleman", Label: "Warm Male Voice"},
		{Value: "Chinese (Mandarin)_Warm_Bestie", Label: "Warm Bestie"},
		{Value: "Chinese (Mandarin)_Male_Announcer", Label: "Male Announcer"},
		{Value: "Chinese (Mandarin)_Sweet_Lady", Label: "Sweet Female Voice"},
		{Value: "Chinese (Mandarin)_Southern_Young_Man", Label: "Southern Young Man"},
		{Value: "Chinese (Mandarin)_Wise_Women", Label: "Wise Sister"},
		{Value: "Chinese (Mandarin)_Gentle_Youth", Label: "Gentle Youth"},
		{Value: "Chinese (Mandarin)_Warm_Girl", Label: "Warm Girl"},
		{Value: "Chinese (Mandarin)_Kind-hearted_Elder", Label: "Kind-hearted Elder"},
		{Value: "Chinese (Mandarin)_Cute_Spirit", Label: "Cute Spirit"},
		{Value: "Chinese (Mandarin)_Radio_Host", Label: "Radio Host Male"},
		{Value: "Chinese (Mandarin)_Lyrical_Voice", Label: "Lyrical Male Voice"},
		{Value: "Chinese (Mandarin)_Straightforward_Boy", Label: "Straightforward Boy"},
		{Value: "Chinese (Mandarin)_Sincere_Adult", Label: "Sincere Youth"},
		{Value: "Chinese (Mandarin)_Gentle_Senior", Label: "Gentle Senior Girl"},
		{Value: "Chinese (Mandarin)_Stubborn_Friend", Label: "Stubborn Childhood Friend"},
		{Value: "Chinese (Mandarin)_Crisp_Girl", Label: "Crisp Girl"},
		{Value: "Chinese (Mandarin)_Pure-hearted_Boy", Label: "Pure-hearted Boy Next Door"},
		{Value: "Chinese (Mandarin)_Soft_Girl", Label: "Soft Girl"},
		// Chinese (Cantonese)
		{Value: "Cantonese_ProfessionalHost（F)", Label: "Professional Female Host"},
		{Value: "Cantonese_GentleLady", Label: "Gentle Female Voice"},
		{Value: "Cantonese_ProfessionalHost（M)", Label: "Professional Male Host"},
		{Value: "Cantonese_PlayfulMan", Label: "Playful Male Voice"},
		{Value: "Cantonese_CuteGirl", Label: "Cute Girl"},
		{Value: "Cantonese_KindWoman", Label: "Kind Female Voice"},
		// English
		{Value: "Santa_Claus", Label: "Santa Claus"},
		{Value: "Grinch", Label: "Grinch"},
		{Value: "Rudolph", Label: "Rudolph"},
		{Value: "Arnold", Label: "Arnold"},
		{Value: "Charming_Santa", Label: "Charming Santa"},
		{Value: "Charming_Lady", Label: "Charming Lady"},
		{Value: "Sweet_Girl", Label: "Sweet Girl"},
		{Value: "Cute_Elf", Label: "Cute Elf"},
		{Value: "Attractive_Girl", Label: "Attractive Girl"},
		{Value: "Serene_Woman", Label: "Serene Woman"},
		{Value: "English_Trustworthy_Man", Label: "Trustworthy Man"},
		{Value: "English_Graceful_Lady", Label: "Graceful Lady"},
		{Value: "English_Aussie_Bloke", Label: "Aussie Bloke"},
		{Value: "English_Whispering_girl", Label: "Whispering girl"},
		{Value: "English_Diligent_Man", Label: "Diligent Man"},
		{Value: "English_Gentle-voiced_man", Label: "Gentle-voiced man"},
	},

	// Aliyun Qwen TTS voice list (base list; model filtering handled by GetAliyunQwenVoicesByModel)
	"aliyun_qwen": {
		{Value: "Cherry", Label: "Qianyue"},
		{Value: "Serena", Label: "Suyao"},
		{Value: "Ethan", Label: "Chenxu"},
		{Value: "Chelsie", Label: "Qianxue"},
		{Value: "Momo", Label: "Motu"},
		{Value: "Vivian", Label: "Shisan"},
		{Value: "Moon", Label: "Yuebai"},
		{Value: "Maia", Label: "Siyue"},
		{Value: "Kai", Label: "Kai"},
		{Value: "Nofish", Label: "Buchiyu"},
		{Value: "Bella", Label: "Mengbao"},
		{Value: "Jennifer", Label: "Jennifer"},
		{Value: "Ryan", Label: "Tiancha"},
	},

	// Xunfei online TTS voice list
	// Note: A set of commonly used static voices is maintained here; actual availability depends on Xunfei console authorization.
	// Reference:
	// https://www.xfyun.cn/doc/tts/online_tts/API.html
	// https://aiui.xfyun.cn/doc/aiui/3_access_service/access_interact/functions/speech_synthesis.html
	"xunfei": {
		{Value: "xiaoyan", Label: "Xiaoyan (female, default recommended)"},
		{Value: "xiaofeng", Label: "Xiaofeng (male)"},
		{Value: "yezi", Label: "Xiaolu (female)"},
		{Value: "yifei", Label: "Yifei (female)"},
		{Value: "yiping", Label: "Yiping (female)"},
		{Value: "qige", Label: "Qige (male)"},
		{Value: "chaoge", Label: "Chaoge (male)"},
		{Value: "pengfei", Label: "Xiaopeng (male)"},
		{Value: "xiaoxin", Label: "Cute Xiaoxin (child)"},
		{Value: "john", Label: "John (English male)"},
		{Value: "catherine", Label: "Catherine (English female)"},
	},

	// Xunfei Super TTS voice list
	// Note: A set of recommended static voices is maintained here; actual availability depends on Xunfei console authorization.
	"xunfei_super_tts": {
		{Value: "x6_lingxiaoxue_pro", Label: "Ling Xiaoxue (x6)"},
		{Value: "x6_lingfeiyi_pro", Label: "Ling Feiyi (x6)"},
		{Value: "x6_lingxiaoli_pro", Label: "Ling Xiaoli (x6)"},
		{Value: "x6_lingxiaoyue_pro", Label: "Ling Xiaoyue (x6)"},
		{Value: "x6_lingxiaoxuan_pro", Label: "Ling Xiaoxuan (x6)"},
		{Value: "x6_lingyuyan_pro", Label: "Ling Yuyan (x6)"},
		{Value: "x6_lingyouyou_pro", Label: "Ling Youyou (x6)"},
		{Value: "x6_feizheChat_pro", Label: "Feizhe Chat (x6)"},
		{Value: "x6_xiaoqiChat_pro", Label: "Xiaoqi Chat (x6)"},
		{Value: "x5_lingxiaotang_flow", Label: "Ling Xiaotang (x5)"},
		{Value: "x5_lingyuzhao_flow", Label: "Ling Yuzhao (x5)"},
		{Value: "x4_zijin_oral", Label: "Zijin (x4, conversational)"},
		{Value: "x4_ziyang_oral", Label: "Ziyang (x4, conversational)"},
	},

	// Zhipu TTS voice list
	"zhipu": {
		{Value: "tongtong", Label: "Tongtong (default voice)"},
		{Value: "chuichui", Label: "Chuichui"},
		{Value: "xiaochen", Label: "Xiaochen"},
		{Value: "jam", Label: "Dongdong Animal Circle jam voice"},
		{Value: "kazi", Label: "Dongdong Animal Circle kazi voice"},
		{Value: "douji", Label: "Dongdong Animal Circle douji voice"},
		{Value: "luodo", Label: "Dongdong Animal Circle luodo voice"},
	},
}

// GetVoiceOptionsByProvider returns the voice list for the given provider.
func GetVoiceOptionsByProvider(provider string) []VoiceOption {
	if voices, ok := VoiceOptions[provider]; ok {
		return voices
	}
	return []VoiceOption{}
}

// GetAliyunQwenVoicesByModel returns the voice list for the given Qwen model name.
// Uses the model mapping in the qwen package to retrieve accurate voice options.
func GetAliyunQwenVoicesByModel(model string) []VoiceOption {
	model = strings.TrimSpace(model)
	if model == "" {
		// Return base list when no model is specified.
		return GetVoiceOptionsByProvider("aliyun_qwen")
	}

	// Use local function to get voices for the given model.
	voices := GetVoicesByModel(model)
	if voices == nil || len(voices) == 0 {
		// Fall back to base list when no voices found for the given model.
		return GetVoiceOptionsByProvider("aliyun_qwen")
	}

	// Convert VoiceInfo to VoiceOption.
	result := make([]VoiceOption, 0, len(voices))
	for _, v := range voices {
		result = append(result, VoiceOption{
			Value: v.Value,
			Label: v.Label,
		})
	}
	return result
}
