<template>
  <cl-form
      :key="JSON.stringify(form)"
      :model="form"
  >
    <cl-form-item :span="4" prop="key" label="Key">
      <el-input v-model="internalForm.key" disabled/>
    </cl-form-item>
    <cl-form-item :span="4" prop="name" label="Name">
      <el-input v-model="internalForm.name" disabled/>
    </cl-form-item>
    <cl-form-item :span="4" prop="description" label="Description">
      <el-input v-model="internalForm.description" type="textarea" disabled/>
    </cl-form-item>
    <cl-form-item :span="4" prop="proxy" label="Proxy">
      <el-input v-model="internalForm.proxy" placeholder="Proxy" @change="onProxyChange"/>
    </cl-form-item>
  </cl-form>
</template>

<script lang="ts">
import {defineComponent, onBeforeMount, ref, watch} from 'vue';

export default defineComponent({
  name: 'DependencySettingForm',
  props: {
    form: {
      type: Object,
      default: () => {
      }
    }
  },
  emits: [
    'change',
  ],
  setup(props, {emit}) {
    const internalForm = ref({});

    const onProxyChange = () => {
      emit('change', internalForm.value);
    };

    onBeforeMount(() => {
      internalForm.value = {...props.form};
    });

    return {
      internalForm,
      onProxyChange,
    };
  },
});
</script>

<style scoped>

</style>
