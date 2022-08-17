# import tensorflow.compat.v1 as tf
from genericpath import exists
import tensorflow
import numpy as np
from pprint import PrettyPrinter
from os import path, mkdir

import sys
sys.path.append('../')  # 将系统路径提高一层

from Simulator.data_model import BoilerDataSet
from sim_rnn_model import RNNSimulatorModel

tf = tensorflow.compat.v1
tf.disable_eager_execution()
# tf.disable_v2_behavior()

# 定义参数，第一个是参数名称，第二个参数是默认值，第三个是参数描述
# 参数有四种取值：整数integer，浮点数float，字符串string，逻辑值boolean

# Data and model checkpoints directories
tf.app.flags.DEFINE_integer("display_iter", 200, "display_iter")
tf.app.flags.DEFINE_integer("save_log_iter", 100, "save_log_iter")

# Model params
tf.app.flags.DEFINE_integer("input_size", 202, "Input size")  # external_input + state + action
tf.app.flags.DEFINE_integer("output_size", 158, "Output size")  # state size

# Optimization
tf.app.flags.DEFINE_integer("num_steps", 10, "Number of steps")
tf.app.flags.DEFINE_float("val_ratio", 0.2, "valid ratio")          # @todo 在大规模数据集上可以改为0.1
tf.app.flags.DEFINE_integer("batch_size", 1, "The size of batch")   # 一个批次上的数据量
tf.app.flags.DEFINE_integer("max_epoch", 50, "Total training epoches")
tf.app.flags.DEFINE_float("grad_clip", 5., "Clip gradients at this value")
tf.app.flags.DEFINE_float("learning_rate", 0.001, "Initial learning rate at early stage. [0.001]")
tf.app.flags.DEFINE_float("learning_rate_decay", 0.95, "Decay rate of learning rate. [0.99]")
tf.app.flags.DEFINE_float("keep_prob", 1, "Keep probability of input data and dropout layer. [0.8]")
tf.app.flags.DEFINE_float("l2_weight", 0.0, "weight of l2 loss")

FLAGS = tf.app.flags.FLAGS

class cell_config(object):
    """ Simulator Cell config """
    # list, [coaler_num_units, burner_num_units, steamer_num_units]
    num_units = [128, 128, 64]   # 各层神经元数量

    # data is [external_input, state(coaler, burner, steamer), action(coaler, burner, steamer)]
    # data size is [11, 147(68, 62, 17), 44(21, 19, 4)]. Total size is 202
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

def fit_and_evaluate(model, train_X, train_y, valid_X, valid_y, learning_rate, epochs=500):
    print("=============== epoch ================")
    early_stopping_cb = tf.keras.callbacks.EarlyStopping(
        monitor="val_mae", patience=50, restore_best_weights=True)
    opt = tf.keras.optimizers.SGD(learning_rate=learning_rate, momentum=0.9)
    model.compile(loss=tf.keras.losses.Huber(), optimizer=opt, metrics=["mae"])
    history = model.fit(x=train_X, 
                    y=train_y, 
                    # validation_data=valid_X, 
                    epochs=epochs,
                    callbacks=[early_stopping_cb])
    # valid_loss, valid_mae = model.evaluate(valid_y)
    # print("loss: {}".format(valid_loss))
    # print("mae : {}".format(valid_mae))
    # return valid_mae * 1e6

def main(_):
    np.random.seed(2022)    # 设置随机种子

    # tf.app.flags.FLAGS.__flags为包含了所有输入的列表
    # 当然，也可以单个查询，格式为：FLAGS.参数名
    # pprint模块负责以合适的格式打印便于阅读的行块。它使用换行和缩进以明确的方式打印数据。
    # PrettyPrinter().pprint(tf.app.flags.FLAGS.__flags)  # 打印参数列表 @debug

    # 设置GPU
    run_config = tf.ConfigProto()
    run_config.gpu_options.allow_growth = True

    # read data 读入数据
    boiler_dataset = BoilerDataSet(num_steps=FLAGS.num_steps, val_ratio=FLAGS.val_ratio)
    train_X, train_y = boiler_dataset.train_X, boiler_dataset.train_y
    valid_X, valid_y = boiler_dataset.valid_X, boiler_dataset.valid_y

    # print dataset info 打印数据信息 @debug
    print('train samples: {0}'.format(len(train_X)))
    print('valid samples: {0}'.format(len(valid_X)))

    # model construction
    tf.reset_default_graph()    # 清除默认图形堆栈并重置全局默认图形
    # rnn_model = RNNSimulatorModel(cell_config=cell_config(), FLAGS=FLAGS)
    # @change
    LSTM_units = 160
    rnn_model = tensorflow.keras.Sequential([
        tensorflow.keras.layers.LSTM(units=LSTM_units, activation='tanh', return_sequences=True, input_shape=[None, FLAGS.input_size]),
        # tensorflow.keras.layers.LSTM(32, return_sequences=True),
        # tensorflow.keras.layers.LSTM(32),
        tensorflow.keras.layers.Dense(FLAGS.output_size)
    ])

    # print trainable params @debug
    for i in tf.trainable_variables():
        print(i)
    
    # count the parameters in our model @debug 显示模型参数个数，仅供提示
    # total_parameters = 0
    # for variable in tf.trainable_variables():
    #     # shape is an array of tf.Dimension
    #     shape = variable.get_shape()
    #     # print(shape)
    #     # print(len(shape))
    #     variable_parameters = 1
    #     for dim in shape:
    #         # print(dim)
    #         variable_parameters *= dim.value
    #     # print(variable_parameters)
    #     total_parameters += variable_parameters
    # print('total parameters: {}'.format(total_parameters))

    # path for log saving   指定保存目录
    model_name = "sim_rnn_lstm_dense"
    # logdir = './logs/{}-{}-{}-{}-{}-{:.2f}-{:.4f}-{:.2f}-{:.5f}/'.format(
    #     model_name, cell_config.num_units[0], cell_config.num_units[1], cell_config.num_units[2],
    #     FLAGS.num_steps, FLAGS.keep_prob, FLAGS.learning_rate, FLAGS.learning_rate_decay, FLAGS.l2_weight)
    # @change
    logdir = './logs/{}-{}-{}-{:.2f}-{:.4f}-{:.2f}-{:.5f}/'.format(
        model_name, LSTM_units, FLAGS.num_steps, FLAGS.keep_prob, FLAGS.learning_rate, FLAGS.learning_rate_decay, FLAGS.l2_weight)
    model_dir = logdir + 'saved_models/'

    # 创建保存结果的文件夹
    if not path.exists('./logs'):
        mkdir('./logs')
    if not path.exists(logdir):
        mkdir(logdir)
    if not path.exists(model_dir):
        mkdir(model_dir)
    # results_dir = logdir + 'results/' # @todo not used

    # 开始训练！
    fit_and_evaluate(rnn_model, train_X, train_y, valid_X, valid_y, learning_rate=FLAGS.learning_rate, epochs=FLAGS.max_epoch)
    # with tf.Session(config=run_config) as sess:
    #     summary_writer = tf.summary.FileWriter(logdir)

    #     sess.run(tf.global_variables_initializer())     # 初始化全局变量
    #     saver = tf.train.Saver()

    #     iter = 0
    #     valid_losses = [np.inf]

    #     for i in range(FLAGS.max_epoch):
    #         print('----------epoch {}-----------'.format(i))
    #         # learning_rate = FLAGS.learning_rate
    #         learning_rate = FLAGS.learning_rate * (   # @todo 加入遗忘
    #             FLAGS.learning_rate_decay ** i
    #         )

    #         for batch_X, batch_y in boiler_dataset.generate_one_epoch(train_X, train_y, FLAGS.batch_size):
    #             iter += 1
    #             train_data_feed = {
    #                 rnn_model.learning_rate: learning_rate,
    #                 rnn_model.keep_prob: FLAGS.keep_prob,
    #                 rnn_model.inputs: batch_X,
    #                 rnn_model.targets: batch_y,
    #             }
    #             train_loss, _, merged_summ = sess.run(
    #                 [rnn_model.loss, rnn_model.train_opt, rnn_model.merged_summ], train_data_feed)
    #             if iter % FLAGS.save_log_iter == 0:
    #                 summary_writer.add_summary(merged_summ, iter)
    #             if iter % FLAGS.display_iter == 0:
    #                 valid_loss = 0
    #                 for val_batch_X, val_batch_y in boiler_dataset.generate_one_epoch(valid_X, valid_y, FLAGS.batch_size):
    #                     val_data_feed = {
    #                         rnn_model.keep_prob: 1.0,
    #                         rnn_model.inputs: val_batch_X,
    #                         rnn_model.targets: val_batch_y,
    #                     }
    #                     batch_loss = sess.run(rnn_model.loss, val_data_feed)
    #                     valid_loss += batch_loss
    #                 num_batches = int(len(valid_X)) // FLAGS.batch_size
    #                 valid_loss /= num_batches
    #                 valid_losses.append(valid_loss)
    #                 valid_loss_sum = tf.Summary(
    #                     value=[tf.Summary.Value(tag="valid_loss", simple_value=valid_loss)])
    #                 summary_writer.add_summary(valid_loss_sum, iter)

    #                 if valid_loss < min(valid_losses[:-1]):
    #                     print('iter {}\tvalid_loss = {:.6f}\tmodel saved!!'.format(
    #                         iter, valid_loss))
    #                     saver.save(sess, model_dir +
    #                                'model_{}.ckpt'.format(iter))
    #                     saver.save(sess, model_dir + 'final_model.ckpt')
    #                 else:
    #                     print('iter {}\tvalid_loss = {:.6f}\t'.format(
    #                         iter, valid_loss))

    # print('stop training !!!')

if __name__ == '__main__':
    #tf.app.run()的作用：先处理flag解析，然后执行main函数
    tf.app.run()