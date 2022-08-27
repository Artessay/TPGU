import os
import numpy as np
import tensorflow as tf


import sys
sys.path.append('../')  # rise system path

from Simulator.data_model import BoilerDataSet
from Simulator.sim_rnn_model import SimulatorRNNModel

class sim_config(object):
    num_steps = 10
    valid_ratio = 0.2   # 0.1 will be better for large dataset

    input_size = 202
    num_neurons = 160
    num_layers = 3
    output_size = 158

    keep_prob = 1
    learning_rate = 0.001
    learning_rate_decay = 0.96
    decay_steps = 5000
    l2_weight = 0

    max_epoch = 50
    batch_size = 1

    save_log_iter = 10
    display_iter = 20

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

def reset_random_seed(seed=2022):
    tf.random.set_seed(seed)
    np.random.seed(seed)

def fit_and_evaluate(model, train_X, train_y, valid_X, valid_y, model_dir, batch_size, epochs=500):
    callback_list = [
        tf.keras.callbacks.EarlyStopping(
            monitor="val_loss", 
            patience=50, 
            restore_best_weights=True
        ),
        tf.keras.callbacks.ModelCheckpoint(
            filepath= os.path.join(model_dir, 'model-{epoch:02d}-{val_loss:.4f}.h5'),
            monitor="val_loss",
            verbose=1,
            save_weights_only=True,
            save_best_only=True,
        ),
        tf.keras.callbacks.TensorBoard(log_dir='./logs/')
    ]   
    
    history = model.fit(
        x=train_X, y=train_y,
        batch_size=batch_size,
        epochs=epochs, 
        validation_data=(valid_X, valid_y),
        callbacks=callback_list)
    # valid_loss, valid_mae = model.evaluate(x=valid_X, y=valid_y) # Returns the loss value & metrics values for the model in test mode
    # return valid_mae * 1e6  # valid mean absolute error

def main():
    """
    use self compiled tensorflow library will improve speed. The command line to build it is:
    bazel build -c opt --copt=-mavx --copt=-mavx2 --copt=-mfma --copt=-mfpmath=both --copt=-msse4.2 --config=cuda -k //tensorflow/tools/pip_package:build_pip_package
    Of course, the code could run in tensorflow 2 without it.
    """

    # set random seed
    reset_random_seed()    

    # get parameters
    num_steps = sim_config.num_steps
    valid_ratio = sim_config.valid_ratio
    batch_size = sim_config.batch_size
    max_epoch = sim_config.max_epoch

    # read data
    boiler_dataset = BoilerDataSet(num_steps=num_steps, val_ratio=valid_ratio)
    train_X, train_y = boiler_dataset.train_X, boiler_dataset.train_y
    valid_X, valid_y = boiler_dataset.valid_X, boiler_dataset.valid_y

    # prepare model
    model = SimulatorRNNModel(sim_config=sim_config())
    model.summary()

    # path for log saving
    model_name = "LSTM"
    logdir = './logs/{}-{}-{}-{}-{}-{:.2f}-{:.4f}-{:.2f}-{:.5f}/'.format(
        model_name, cell_config.num_units[0], cell_config.num_units[1], cell_config.num_units[2],
        sim_config.num_steps, sim_config.keep_prob, sim_config.learning_rate, sim_config.learning_rate_decay, sim_config.l2_weight)
    model_dir = logdir + 'saved_models/'
    results_dir = logdir + 'results/'

    # create folders
    if not os.path.exists('./logs'):
        os.mkdir('./logs')
    if not os.path.exists(logdir):
        os.mkdir(logdir)
    if not os.path.exists(model_dir):
        os.mkdir(model_dir)
    if not os.path.exists(results_dir):
        os.mkdir(results_dir)
    
    # train model
    fit_and_evaluate(model, train_X, train_y, valid_X, valid_y, model_dir, batch_size, max_epoch)
    model.save(results_dir)

if __name__ == '__main__':
    main()