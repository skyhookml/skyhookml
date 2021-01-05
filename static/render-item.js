export default {
	props: ['dataType', 'item'],
	template: `
<div>
	<template v-if="dataType == 'video'">
		<video controls :src="'/items/'+item.ID+'/get?format=mp4'"></video>
	</template>
	<template v-else-if="dataType == 'image'">
		<img :src="'/items/'+item.ID+'/get?format=jpeg'" />
	</template>
</div>
	`,
};
