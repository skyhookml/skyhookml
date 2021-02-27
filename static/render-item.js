export default {
	props: ['dataType', 'item'],
	template: `
<div>
	<template v-if="dataType == 'video'">
		<video controls :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=mp4'" class="explore-result-img"></video>
	</template>
	<template v-else-if="dataType == 'image'">
		<img :src="'/datasets/'+item.Dataset.ID+'/items/'+item.Key+'/get?format=jpeg'" class="explore-result-img" />
	</template>
</div>
	`,
};
