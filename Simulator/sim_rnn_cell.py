import tensorflow as tf
import numpy as np
# import collections
# import hashlib
# import numbers

from tensorflow.python.eager import context
from tensorflow.python.framework import constant_op
from tensorflow.python.framework import ops
from tensorflow.python.framework import dtypes
from tensorflow.python.layers import base as base_layer
from tensorflow.contrib.rnn import RNNCell
from tensorflow.contrib.rnn import LSTMCell
from tensorflow.contrib.rnn import GRUCell
from tensorflow.python.ops import array_ops
# from tensorflow.python.ops import clip_ops
from tensorflow.python.ops import init_ops
from tensorflow.python.ops import math_ops
from tensorflow.python.ops import nn_ops
# from tensorflow.python.ops import partitioned_variables
# from tensorflow.python.ops import random_ops
# from tensorflow.python.ops import tensor_array_ops
# from tensorflow.python.ops import variable_scope as vs
# from tensorflow.python.ops import variables as tf_variables
from tensorflow.python.ops.rnn_cell_impl import LSTMStateTuple
# from tensorflow.python.util import nest
# from tensorflow.python.util.tf_export import tf_export
from tensorflow.python.ops.rnn_cell_impl import _zero_state_tensors


class _LayerRNNCell(RNNCell):
  """Subclass of RNNCells that act like proper `tf.Layer` objects.

  For backwards compatibility purposes, most `RNNCell` instances allow their
  `call` methods to instantiate variables via `tf.get_variable`.  The underlying
  variable scope thus keeps track of any variables, and returning cached
  versions.  This is atypical of `tf.layer` objects, which separate this
  part of layer building into a `build` method that is only called once.

  Here we provide a subclass for `RNNCell` objects that act exactly as
  `Layer` objects do.  They must provide a `build` method and their
  `call` methods do not access Variables `tf.get_variable`.
  """

  def __call__(self, inputs, state, scope=None, *args, **kwargs):
    """Run this RNN cell on inputs, starting from the given state.

    Args:
      inputs: `2-D` tensor with shape `[batch_size, input_size]`.
      state: if `self.state_size` is an integer, this should be a `2-D Tensor`
        with shape `[batch_size, self.state_size]`.  Otherwise, if
        `self.state_size` is a tuple of integers, this should be a tuple
        with shapes `[batch_size, s] for s in self.state_size`.
      scope: optional cell scope.
      *args: Additional positional arguments.
      **kwargs: Additional keyword arguments.

    Returns:
      A pair containing:

      - Output: A `2-D` tensor with shape `[batch_size, self.output_size]`.
      - New state: Either a single `2-D` tensor, or a tuple of tensors matching
        the arity and shapes of `state`.
    """
    # Bypass RNNCell's variable capturing semantics for LayerRNNCell.
    # Instead, it is up to subclasses to provide a proper build
    # method.  See the class docstring for more details.
    return base_layer.Layer.__call__(self, inputs, state, scope=scope,
                                     *args, **kwargs)


class SimulatorRNNCell(_LayerRNNCell):
    """
    coaler RNN: (external_input_t, coaler_hidden_t-1 , coaler_action_t) --> (coaler_hidden_t, coaler_cell_t)
    burner RNN: (coaler_hidden_t, burner_hidden_t-1 , burner_action_t) --> (burner_hidden_t, burner_cell_t)
    steamer RNN: (burner_hidden_t, steamer_hidden_t-1 , steamer_action_t) --> (steamer_hidden_t, steamer_cell_t)

    loss: sum of three parts
    part1: coaler_hidden_t, coaler_state_t
    part2: burner_hidden_t, burner_state_t
    part3: steamer_hidden_t, steamer_state_t
    """
    def __init__(self, cell_config,
                 keep_prob,
                 forget_bias=1.0,
                 activation=None,
                 reuse=None,
                 name=None):
        """
        Args:
          cell_config: simulator config
          num_units: list, [coaler_num_units, burner_num_units, steamer_num_units]
          【NOTE: We add forget_bias (default: 1) to the biases of the forget gate in order to
            reduce the scale of forgetting at the beginning of the training.】
            对reuse另一种解释: (optional) Python boolean describing whether to reuse variables
            in an existing scope.  If not `True`, and the existing scope already has
            the given variables, an error is raised.
            name: String, the name of the layer. Layers with the same name will
            share weights, but to avoid mistakes we require reuse=True in such
            cases.
        """
        # 复习：子类重写父类的init又想调用调用父类的init：super(子类名, self).__init__(...)
        # reuse意思是定义的Layer可以共享参数（即两层的参数一样）
        # name是啥（可能不重要？）
        super(SimulatorRNNCell, self).__init__(_reuse=reuse, name=name)
        # 这是用来检查输入的类型的，Inputs must be 2-dimensional.
        self.input_spec = base_layer.InputSpec(ndim=2)

        self._external_state_pos = cell_config.external_state_pos
        self._coaler_state_pos = cell_config.coaler_state_pos
        self._coaler_action_pos = cell_config.coaler_action_pos
        self._burner_state_pos = cell_config.burner_state_pos
        self._burner_action_pos = cell_config.burner_action_pos
        self._steamer_state_pos = cell_config.steamer_state_pos
        self._steamer_action_pos = cell_config.steamer_action_pos

        self._external_state_size = cell_config.external_state_size
        self._coaler_state_size = cell_config.coaler_state_size
        self._coaler_action_size = cell_config.coaler_action_size
        self._burner_state_size = cell_config.burner_state_size
        self._burner_action_size = cell_config.burner_action_size
        self._steamer_state_size = cell_config.steamer_state_size
        self._steamer_action_size = cell_config.steamer_action_size

        # num_units: list, [coaler_num_units, burner_num_units, steamer_num_units]
        _num_units = cell_config.num_units  # TODO
        self._coaler_num_units = _num_units[0]
        self._burner_num_units = _num_units[1]
        self._steamer_num_units = _num_units[2]
        self._forget_bias = forget_bias
        self._activation = activation or math_ops.tanh # 激活函数
        self._input_keep_prob = self._output_keep_prob = keep_prob # input和output的keep_prob一样

    @property
    def state_size(self):
        # 这个state_size只给出了call函数中最后的new_h, new_c的列数，没给行数（行数就是batch_size）
        c_tuple = tuple((self._coaler_num_units, self._burner_num_units, self._steamer_num_units))
        h_tuple = tuple((self._coaler_num_units, self._burner_num_units, self._steamer_num_units))
        return LSTMStateTuple(c_tuple, h_tuple)

    @property
    def output_size(self):
        return tuple((self._coaler_num_units, self._burner_num_units, self._steamer_num_units))

    def get_coaler_inputs(self, inputs):
        # coaler inputs contains external_input, coaler_state and coaler_action
        # inputs: (batch_size, feature_nums)

        # tf.slice的使用tf.slice(inputs,begin,size,name='')，从inputs中抽取部分内容
        # inputs：可以是list,array,tensor
        # begin：n维列表，begin[i] 表示从inputs中第i维抽取数据时，相对0的起始偏移量，
        #       也就是从第i维的begin[i]位置开始抽取数据
        # size：n维列表，size[i]表示要抽取的第i维元素的数目，-1表示一直到底
        external_input = tf.slice(inputs, [0, self._external_state_pos],
                                  [-1, self._external_state_size])

        coaler_state = tf.slice(inputs, [0, self._coaler_state_pos],
                                [-1, self._coaler_state_size])
        coaler_action = tf.slice(inputs, [0, self._coaler_action_pos],
                                 [-1, self._coaler_action_size])
        # tf.concat的使用tf.concat([tensor1, tensor2, tensor3,...], axis)
        # axis表示在第几维度进行拼接（其他维度的len保持不变）
        # 最后的shape就是
        # (batch_size, external_input_feature_nums + coaler_state_feature_nums + coaler_action_feature_nums)
        return tf.concat([external_input, coaler_state, coaler_action], axis=1)

    def get_burner_inputs(self, inputs):
        # burner inputs contains burner_state and burner_action
        # input: (batch_size, feature_nums)
        burner_state = tf.slice(inputs, [0, self._burner_state_pos],
                                [-1, self._burner_state_size])
        burner_action = tf.slice(inputs, [0, self._burner_action_pos],
                                 [-1, self._burner_action_size])
        return tf.concat([burner_state, burner_action], axis=1)

    def get_steamer_inputs(self, inputs):
        # steamer inputs contains steamer_state and steamer_action
        # input: (batch_size, feature_nums)
        steamer_state = tf.slice(inputs, [0, self._steamer_state_pos],
                                 [-1, self._steamer_state_size])
        steamer_action = tf.slice(inputs, [0, self._steamer_action_pos],
                                  [-1, self._steamer_action_size])
        return tf.concat([steamer_state, steamer_action], axis=1)

    # 官网对__init__和build的解释：
    # __init__()，您可以在其中执行所有与输入无关的初始化
    # build()，您可以在其中了解输入张量的形状，并可以执行其余的初始化
    # call()，在那里进行正向计算。
    # 请注意，您不必等到调用 build 来创建变量，您也可以在 __init__中创建它们。
    # 但是，在 build 中创建它们的好处是，它支持根据将要操作的层的输入形状，
    # 创建后期变量。另一方面，在 __init__ 中创建变量意味着需要明确指定创建变量所需的形状。
    def build(self, inputs_shape):
        print("\n\n----------------------------build!!!!!!-----------------------------\n\n")
        # coaler
        # external_...和coaler_...是按照磨煤阶段来分类
        external_input_depth = self._external_state_size
        coaler_input_depth = self._coaler_state_size + self._coaler_action_size
        self._coaler_kernel = self.add_variable( # 卷积核（矩阵乘法中位于右侧）
            "coaler_kernel", # 变量名
            shape=[external_input_depth + coaler_input_depth + self._coaler_num_units, 4 * self._coaler_num_units],
            initializer=orthogonal_lstm_initializer())
        self._coaler_bias = self.add_variable(  # 斜率
            "coaler_bias",
            shape=[4 * self._coaler_num_units],
            initializer=init_ops.zeros_initializer(dtype=self.dtype))
        # burner
        burner_input_depth = self._burner_state_size + self._burner_action_size
        self._burner_kernel = self.add_variable(
            "burner_kernel",
            shape=[burner_input_depth + self._burner_num_units + self._coaler_num_units, 4 * self._burner_num_units],
            initializer=orthogonal_lstm_initializer())
        self._burner_bias = self.add_variable(
            "burner_bias",
            shape=[4 * self._burner_num_units],
            initializer=init_ops.zeros_initializer(dtype=self.dtype))
        # steamer
        steamer_input_depth = self._steamer_state_size + self._steamer_action_size
        self._steamer_kernel = self.add_variable(
            "steamer_kernel",
            shape=[steamer_input_depth + self._steamer_num_units + self._burner_num_units, 4 * self._steamer_num_units],
            initializer=orthogonal_lstm_initializer())
        self._steamer_bias = self.add_variable(
            "steamer_bias",
            shape=[4 * self._steamer_num_units],
            initializer=init_ops.zeros_initializer(dtype=self.dtype))

        self.built = True

    def zero_state(self, batch_size, dtype):
        """Return zero-filled state tensor(s).

        Args:
          batch_size: int, float, or unit Tensor representing the batch size.
          dtype: the data type to use for the state.

        Returns:
          If `state_size` is an int or TensorShape, then the return value is a
          `N-D` tensor of shape `[batch_size, state_size]` filled with zeros.

          If `state_size` is a nested list or tuple, then the return value is
          a nested list or tuple (of the same structure) of `2-D` tensors with
          the shapes `[batch_size, s]` for each s in `state_size`.
        """
        # Try to use the last cached zero_state. This is done to avoid recreating
        # zeros, especially when eager execution is enabled.
        state_size = self.state_size
        is_eager = context.in_eager_mode()
        if is_eager and hasattr(self, "_last_zero_state"):
            (last_state_size, last_batch_size, last_dtype,
             last_output) = getattr(self, "_last_zero_state")
            if (last_batch_size == batch_size and
                last_dtype == dtype and
                last_state_size == state_size):
                return last_output
        with ops.name_scope(type(self).__name__ + "ZeroState", values=[batch_size]):
            output = _zero_state_tensors(state_size, batch_size, dtype)
        if is_eager:
            self._last_zero_state = (state_size, batch_size, dtype, output)
        # output是一个二元tuple，每一个元素又是一个tuple，
        # 再往里才是一个个形状为[batch_size, state_size]的tensor
        return output

    def call(self, inputs, state):
        print("\n\n----------------------------call!!!!!!-----------------------------\n\n")
        # inputs.shape=(batch_size, self.num_steps, self.input_size)
        # 【如果inputs如上面所说，那接下来都矛盾了。】（下面似乎认为inputs.shape=(batch_size, self.input_size)）
        # state: (c, h) is a 3-D tensor
        # c: (c_coaler, c_burner, c_steamer)
        # h: (h_coaler, h_burner, h_steamer)
        # self._state_is_tuple is True for simplicity
        def _should_dropout(p):
            return (not isinstance(p, float)) or p < 1

        # input dropout
        # 在训练时随机使p*100%的feature detectors不起作用，其他的变成原来的1/(1-rate)倍，防止过拟合
        # dropout的工作机制：比如
        """
        x = tf.Variable([[1, 2, 3],
                 [4, 5, 6],
                 [7, 8, 9],
                 [10, 11, 12]], dtype=tf.float32)
        keep_prob = 0.5
        a = tf.nn.dropout(x, keep_prob)
        print(a)
        >>> tf.Tensor([[ 2.  4.  6.]
                    [ 8.  0. 12.]
                    [ 0.  0. 18.]
                    [ 0. 22. 24.]], shape=(4, 3), dtype=float32)

        """

        # 先对整个的inputs drop了一轮，接下来【除了coaler以外（为什么这样做？）】每次都会drop一次
        if _should_dropout(self._input_keep_prob):
            inputs = nn_ops.dropout(inputs, keep_prob=self._input_keep_prob)


        # 得到一个batch的input
        # coaler_inputs.shape=(batch_size, external_state_size + coaler_state_size + coaler_action_size)
        # burner_inputs.shape=(batch_size, burner_state_size + burner_action_size)
        # steamer_inputs.shape=(batch_size, steamer_state_size + steamer_action_size)
        coaler_inputs = self.get_coaler_inputs(inputs)
        burner_inputs = self.get_burner_inputs(inputs)
        steamer_inputs = self.get_steamer_inputs(inputs)

        sigmoid = math_ops.sigmoid
        one = constant_op.constant(1, dtype=dtypes.int32)

        # coaler_h.shape=(batch_size, _coaler_num_units)
        c, h = state
        coaler_h, burner_h, steamer_h = h
        coaler_c, burner_c, steamer_c = c

        # coal mill model
        # with上下文变量管理，作用？
        with tf.compat.v1.variable_scope('coaler'):
            # inputs = self.batch_normalization(inputs, 'coal_mill_bn')
            # matmul是普通的矩阵乘法
            # 这里左矩阵的shape=(batch_size, external_input_depth + coaler_input_depth + self._coaler_num_units)
            # 右矩阵的shape=(external_input_depth + coaler_input_depth + self._coaler_num_units, 4 * _coaler_num_units)
            # coaler_gate_inputs.shape==(batch_size, 4 * _coaler_num_units)
            # 注意bias_add是一个二维矩阵每一行都加一个一维的行向量
            coaler_gate_inputs = math_ops.matmul( 
                array_ops.concat([coaler_inputs, coaler_h], 1), self._coaler_kernel)
            coaler_gate_inputs = nn_ops.bias_add(coaler_gate_inputs, self._coaler_bias)

            # 通过前面的拼接、矩阵乘法一次性把z（这里对应coaler_j）,z^i,z^f,z^o算好（效率高），现在把计算完的结果拆成四个小矩阵
            # shape==(batch_size, _coaler_num_units)
            coaler_i, coaler_j, coaler_f, coaler_o = array_ops.split(
                value=coaler_gate_inputs, num_or_size_splits=4, axis=one)

            coaler_forget_bias_tensor = constant_op.constant(self._forget_bias, dtype=coaler_f.dtype)
            # Note that using `add` and `multiply` instead of `+` and `*` gives a
            # performance improvement. So using those at the cost of readability.
            add = math_ops.add # 注意不是矩阵加法，是一个tensor和一个标量（shape=()）相加
            multiply = math_ops.multiply # 注意这里不是矩阵乘法，是Hadamard Product，逐元素相乘
            coaler_new_c = add(multiply(coaler_c, sigmoid(add(coaler_f, coaler_forget_bias_tensor))),
                               multiply(sigmoid(coaler_i), self._activation(coaler_j))) # shape==(batch_size, _coaler_num_units)
            coaler_new_h = multiply(self._activation(coaler_new_c), sigmoid(coaler_o))  # shape==(batch_size, _coaler_num_units)

        with tf.compat.v1.variable_scope('burner'):
            # inputs = self.batch_normalization(inputs, 'coal_mill_bn')
            # only dropout coaler output
            # 【注意利用的是更新前的coaler_h! 但这似乎和论文的不符？论文是把h_c^t传下去而不是h_c^{t-1}？？？】
            if _should_dropout(self._output_keep_prob):
                coaler_h = nn_ops.dropout(coaler_h, keep_prob=self._output_keep_prob)

            # 这里左矩阵的shape=(batch_size, burner_input_depth + self._burner_num_units + self._coaler_num_units)
            # 右矩阵的shape=(burner_input_depth + self._burner_num_units + self._coaler_num_units, 4 * self._burner_num_units)
            # burner_gate_inputs.shape == (batch_size, 4 * self._burner_num_units)
            burner_gate_inputs = math_ops.matmul( # 这里concat的东西变成了三个，多了一个上一个流程的hidden layer
                array_ops.concat([burner_inputs, burner_h, coaler_h], 1), self._burner_kernel)
            burner_gate_inputs = nn_ops.bias_add(burner_gate_inputs, self._burner_bias)

            burner_i, burner_j, burner_f, burner_o = array_ops.split(
                value=burner_gate_inputs, num_or_size_splits=4, axis=one)

            burner_forget_bias_tensor = constant_op.constant(self._forget_bias, dtype=burner_f.dtype)
            # Note that using `add` and `multiply` instead of `+` and `*` gives a
            # performance improvement. So using those at the cost of readability.
            add = math_ops.add
            multiply = math_ops.multiply
            burner_new_c = add(multiply(burner_c, sigmoid(add(burner_f, burner_forget_bias_tensor))),
                               multiply(sigmoid(burner_i), self._activation(burner_j)))
            burner_new_h = multiply(self._activation(burner_new_c), sigmoid(burner_o))

        with tf.compat.v1.variable_scope('steamer'):
            # inputs = self.batch_normalization(inputs, 'coal_mill_bn')
            # only dropout burner output
            if _should_dropout(self._output_keep_prob):
                burner_h = nn_ops.dropout(burner_h, keep_prob=self._output_keep_prob)

            steamer_gate_inputs = math_ops.matmul(
                array_ops.concat([steamer_inputs, steamer_h, burner_h], 1), self._steamer_kernel)
            steamer_gate_inputs = nn_ops.bias_add(steamer_gate_inputs, self._steamer_bias)

            steamer_i, steamer_j, steamer_f, steamer_o = array_ops.split(
                value=steamer_gate_inputs, num_or_size_splits=4, axis=one)

            steamer_forget_bias_tensor = constant_op.constant(self._forget_bias, dtype=steamer_f.dtype)
            # Note that using `add` and `multiply` instead of `+` and `*` gives a
            # performance improvement. So using those at the cost of readability.
            add = math_ops.add
            multiply = math_ops.multiply
            steamer_new_c = add(multiply(steamer_c, sigmoid(add(steamer_f, steamer_forget_bias_tensor))),
                                multiply(sigmoid(steamer_i), self._activation(steamer_j)))
            steamer_new_h = multiply(self._activation(steamer_new_c), sigmoid(steamer_o))

        new_c = tuple((coaler_new_c, burner_new_c, steamer_new_c))
        new_h = tuple((coaler_new_h, burner_new_h, steamer_new_h))
        # concat_h = array_ops.concat([coaler_new_h, burner_new_h, steamer_new_h], axis=1)
        new_state = LSTMStateTuple(new_c, new_h) # 就是一个tuple，只是给它起了个别名
        return new_h, new_state


def orthogonal_lstm_initializer():
    def orthogonal(shape, dtype=tf.float32, partition_info=None): # 正交规范化？好像有助于缓解梯度爆炸或消失
        # taken from https://github.com/cooijmanstim/recurrent-batch-normalization
        # taken from https://gist.github.com/kastnerkyle/f7464d98fe8ca14f2a1a
        """ benanne lasagne ortho init (faster than qr approach)"""
        # 比如把一个shape=(2,3,4)的一个tensor拍扁，就变成shape=(2,12)
        flat_shape = (shape[0], np.prod(shape[1:]))
        a = np.random.normal(0.0, 1.0, flat_shape) # 就是N(0, 1^2)=N(0,1)的高斯分布
        u, _, v = np.linalg.svd(a, full_matrices=False) # 奇异值分解?
        q = u if u.shape == flat_shape else v  # pick the one with the correct shape
        q = q.reshape(shape)
        return tf.constant(q[:shape[0], :shape[1]], dtype)
    return orthogonal


