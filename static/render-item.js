export default {
	props: ['dataType', 'item'],
	template: `
<div>
	<template v-if="dataType == 'video'">
		<video controls :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=mp4'"></video>
	</template>
	<template v-else-if="dataType == 'image'">
		<img :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=jpeg'" />
	</template>
</div>
	`,
};
