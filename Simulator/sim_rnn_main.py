import os
import numpy as np
import pprint
import tensorflow as tf
import tensorflow.contrib.slim as slim
# import tf_slim as slim #因为tensorflow 2.x没有contrib，所以用这个
import sys
# 添加环境变量，这样才能引入下面两个自己写的模块
sys.path.append('../')

from simrnn_model import RNNSimulatorModel
from data_model import BoilerDataSet
# flags是一个指针，也就是后面实际上调用的是tf.app.flags.DEFINE_integer()
# 如果是tensorflow 2.x的版本，要改成tf.compat.v1.app.flags，因为2.x已经没有.app了
# 下面这些是默认参数，可以在命令行里改成自定义的，
# 如python simrnn_main.py --display_iter 100 --save_log_iter 50
flags = tf.app.flags
# Data and model checkpcheckpointsoints directories
flags.DEFINE_integer("display_iter", 20, "display_iter")
flags.DEFINE_integer("save_log_iter", 10, "save_log_iter")
# Model params
flags.DEFINE_integer("input_size", 202, "Input size")  # external_input + state + action
flags.DEFINE_integer("output_size", 147, "Output size")  # state size
# Optimization
flags.DEFINE_integer("num_steps", 10, "Num of steps")
flags.DEFINE_integer("batch_size", 1, "The size of batch")
flags.DEFINE_integer("max_epoch", 50, "Total training epoches")
flags.DEFINE_float("grad_clip", 5., "Clip gradients at this value") 
# 梯度截断，在训练模型的过程中，我们有可能发生梯度爆炸的情况，这样会导致我们模型训练的失败。
# 我们可以采取一个简单的策略来避免梯度的爆炸，那就是梯度截断Clip, 将梯度约束在某一个区间之内，
# 在训练的过程中，在优化器更新之前进行梯度截断操作。
# 大于grad_clip的梯度，将被修改等于grad_clip
flags.DEFINE_float("learning_rate", 0.001, "Initial learning rate at early stage. [0.001]")
flags.DEFINE_float("learning_rate_decay", 0.95, "Decay rate of learning rate. [0.99]")
flags.DEFINE_float("keep_prob", 1, "Keep probability of input data and dropout layer. [0.8]")
flags.DEFINE_float("l2_weight", 0.0, "weight of l2 loss") 
# L2 loss就是(f(x) - Y)^2，对于大多数CNN网络，我们一般就是使用L2-loss而不是L1-loss，因为L2-loss的收敛速度要比L1-loss要快得多。

# 比如说FLAGS.num_steps可以查到上面定义的num_steps
FLAGS = flags.FLAGS


class cell_config(object):
    """ Simulator Cell config """
    # list, [coaler_num_units, burner_num_units, steamer_num_units]
    # num_units是神经元数量
    num_units = [256, 128, 128] # 暂时先翻倍

    # data is [external_input, state(coaler, burner, steamer), action(coaler, burner, steamer)]
    """
    解读上面的注释：
    一张表（或者叫inputs）长这样：
    ex0     ex1 ...ex10     co0     co1 ...co24     bu0     bu1  ...bu6     ......
    ...（有很多行数据，按照【时间戳】排序！）
   
    其中ex为external_state，co为coaler_state，bu为burner_state，后面被省略的列依次为steamer_state，
    coaler_action，burner_action，steamer_action
    """
    # _pos，就是和表格的“列”相关的位置 _size是有多少列
    # 下一个的_pos = 上一个的_pos + 上一个的_size
    # 可以联系上面input_size, output_size的参数定义的注释，
    # 所有这些直接写出来的数字加起来是input_size=109（就是109列），
    # 因为只输出state，则output_size=47=25+7+15
    external_state_pos = 0
    external_state_size = 11
    coaler_state_pos = external_state_pos + external_state_size
    coaler_state_size = 68
    burner_state_pos = coaler_state_pos + coaler_state_size
    burner_state_size = 62
    steamer_state_pos = burner_state_pos + burner_state_size
    steamer_state_size = 17
    coaler_action_pos = steamer_state_pos + steamer_state_size
    coaler_action_size = 21
    burner_action_pos = coaler_action_pos + coaler_action_size
    burner_action_size = 19
    steamer_action_pos = burner_action_pos + burner_action_size
    steamer_action_size = 4

# 为了打印格式好看
pp = pprint.PrettyPrinter()

# 创建logs文件夹
if not os.path.exists("logs"):
    os.mkdir("logs")

# 展示所有的变量
def show_all_variables():
    # 仅可以查看可训练的变量，即trainable=True的变量
    model_vars = tf.trainable_variables()
    # 展示变量
    slim.model_analyzer.analyze_vars(model_vars, print_info=True)


def main(_):
    # 无法执行sess.run()的原因是tensorflow版本不同导致的，tensorflow版本2.0无法兼容版本1.0.所以需要加这一句
    tf.compat.v1.disable_eager_execution() 
    # 每次用np.random.rand()产生的随机数都是一样的，因为设置了种子
    np.random.seed(2019)

    # 把所有的flag参数都打出来
    # pp.pprint(flags.FLAGS.__flags)

    # 控制显存的占用
    # os.environ["CUDA_VISIBLE_DEVICES"] = "0" # 使用GPU 0 如果=右边改成"0,1"则是使用GPU 0,1
    run_config = tf.compat.v1.ConfigProto()
    run_config.gpu_options.allow_growth = True #动态申请显存，如果没有这个设置，将默认把所有的显存都使用

    # read data
    boiler_dataset = BoilerDataSet(num_steps=FLAGS.num_steps, output_size=FLAGS.output_size)
    train_X = boiler_dataset.train_X
    train_y = boiler_dataset.train_y
    val_X = boiler_dataset.val_X
    val_y = boiler_dataset.val_y
    # print dataset info
    num_train = len(train_X)
    num_valid = len(val_X)
    print('train samples: {0}'.format(num_train))
    print('eval samples: {0}'.format(num_valid))
    print('train_X.shape=', train_X.shape, ', train_y.shape=', train_y.shape)
    print('val_X.shape=', val_X.shape, ', val_y.shape=', val_y.shape)
    # model construction
    # reset_default_graph简介见https://blog.csdn.net/duanlianvip/article/details/98626111（不重要）
    # 补充知识：[python计算图](https://zhuanlan.zhihu.com/p/344846077)
    # [TensorFlow基础知识:计算图中的Op,边,和张量](https://blog.csdn.net/u014281392/article/details/73849199)
    tf.compat.v1.reset_default_graph()
    rnn_model = RNNSimulatorModel(cell_config(), FLAGS)

    # count the parameters in our model
    total_parameters = 0
    for variable in tf.compat.v1.trainable_variables():
        # shape is an array of tf.Dimension
        shape = variable.get_shape()
        print(variable,": ",shape)
        # print(len(shape))
        variable_parameters = 1
        for dim in shape:
            # print(dim)
            variable_parameters *= dim.value
        # print(variable_parameters)
        total_parameters += variable_parameters
    print('total parameters: {}'.format(total_parameters))

    # path for log saving
    model_name = "sim_rnn"
    logdir = './logs/{}-{}-{}-{}-{}-{:.2f}-{:.4f}-{:.2f}-{:.5f}/'.format(
        model_name, cell_config.num_units[0], cell_config.num_units[1], cell_config.num_units[2],
        FLAGS.num_steps, FLAGS.keep_prob, FLAGS.learning_rate, FLAGS.learning_rate_decay, FLAGS.l2_weight)
    model_dir = logdir + 'saved_models/'

    if not os.path.exists(logdir):
        os.mkdir(logdir)
    if not os.path.exists(model_dir):
        os.mkdir(model_dir)
    results_dir = logdir + 'results/'

    with tf.compat.v1.Session(config=run_config) as sess:
        summary_writer = tf.compat.v1.summary.FileWriter(logdir)

        sess.run(tf.compat.v1.global_variables_initializer())
        saver = tf.compat.v1.train.Saver()

        iter = 0
        valid_losses = [np.inf]

        # start training!
        for i in range(FLAGS.max_epoch):
            print('----------epoch {}-----------'.format(i))
            # learning_rate = FLAGS.learning_rate
            learning_rate = FLAGS.learning_rate * (
                FLAGS.learning_rate_decay ** i
            ) # learning_rate每一轮都在减小，指数级减小

            for batch_X, batch_y in boiler_dataset.generate_one_epoch(train_X, train_y, FLAGS.batch_size):
                # batch_X.shape=(batch_size, self.num_steps, table_col)
                # batch_y.shape=(batch_size, table_col2)
                iter += 1
                train_data_feed = {
                    rnn_model.learning_rate: learning_rate,
                    rnn_model.keep_prob: FLAGS.keep_prob,
                    rnn_model.inputs: batch_X,
                    rnn_model.targets: batch_y,
                }
                # 把数据feed到rnn_model的四个placeholder里面计算
                # sess.run()第一个参数是计算什么东西，这里是计算rnn_model中的这三个参数，第二个是喂什么东西进去
                train_loss, _, merged_summ = sess.run(
                    [rnn_model.loss, rnn_model.train_opt, rnn_model.merged_summ], train_data_feed)
                if iter % FLAGS.save_log_iter == 0:
                    summary_writer.add_summary(merged_summ, iter)
                # 交叉验证（一般用于数据量不是很充足的时候，比如少于10000条）
                if iter % FLAGS.display_iter == 0:
                    valid_loss = 0
                    for val_batch_X, val_batch_y in boiler_dataset.generate_one_epoch(val_X, val_y, FLAGS.batch_size):
                        val_data_feed = {
                            rnn_model.keep_prob: 1.0, # 不丢数据，因为是做验证
                            rnn_model.inputs: val_batch_X,
                            rnn_model.targets: val_batch_y,
                        }
                        batch_loss = sess.run(rnn_model.loss, val_data_feed)
                        valid_loss += batch_loss
                    num_batches = int(len(val_X)) // FLAGS.batch_size
                    valid_loss /= num_batches
                    valid_losses.append(valid_loss)
                    valid_loss_sum = tf.Summary(
                        value=[tf.Summary.Value(tag="valid_loss", simple_value=valid_loss)])
                    summary_writer.add_summary(valid_loss_sum, iter)

                    if valid_loss < min(valid_losses[:-1]):
                        print('iter {}\tvalid_loss = {:.6f}\tmodel saved!!'.format(
                            iter, valid_loss))
                        saver.save(sess, model_dir +
                                   'model_{}.ckpt'.format(iter))
                        saver.save(sess, model_dir + 'final_model.ckpt')
                    else:
                        print('iter {}\tvalid_loss = {:.6f}\t'.format(
                            iter, valid_loss))
                # end of --- if iter % FLAGS.display_iter == 0:
            # end of --- for batch_X, batch_y in boiler_dataset.generate_one_epoch(train_X, train_y, FLAGS.batch_size):
        # end of --- for i in range(FLAGS.max_epoch)
    print('stop training !!!')

# 使用这种方式保证了，如果此文件被其他文件 import的时候，不会执行main 函数
if __name__ == '__main__':
    tf.compat.v1.app.run() # 解析命令行参数，调用main 函数 main(sys.argv)